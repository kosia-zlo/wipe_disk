package wipe

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"wipedisk_enterprise/internal/logging"
	"wipedisk_enterprise/internal/system"
)

// PersistentFileConfig конфигурация для затирания через постоянный файл
type PersistentFileConfig struct {
	BufferSize  int64         // Размер буфера в байтах (1МБ по умолчанию)
	MaxDuration time.Duration // Максимальная длительность операции
	Progress    chan<- ProgressInfo
	Logger      *logging.EnterpriseLogger
	Pattern     []byte // Паттерн для записи (nil = случайные данные)
}

// PersistentFileWiper реализует затирание через один постоянный файл
type PersistentFileWiper struct {
	config *PersistentFileConfig
	mu     sync.Mutex
}

// NewPersistentFileWiper создает новый экземпляр PersistentFileWiper
func NewPersistentFileWiper(config *PersistentFileConfig) *PersistentFileWiper {
	if config.BufferSize <= 0 {
		config.BufferSize = 1024 * 1024 // 1МБ по умолчанию
	}
	return &PersistentFileWiper{config: config}
}

// Wipe выполняет затирание свободного места на указанном диске
func (pfw *PersistentFileWiper) Wipe(ctx context.Context, drivePath string) (*WipeResult, error) {
	result := &WipeResult{}
	startTime := time.Now()

	// Нормализация пути диска
	drivePath = filepath.Clean(drivePath)
	if len(drivePath) != 2 || drivePath[1] != ':' {
		return nil, fmt.Errorf("некорректный путь к диску: %s", drivePath)
	}

	// Проверка доступности диска
	diskInfo, err := system.GetDiskInfoForPath(drivePath)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения информации о диске: %w", err)
	}

	if diskInfo.FreeSize == 0 {
		return nil, fmt.Errorf("на диске нет свободного места")
	}

	pfw.config.Logger.Log("INFO", "Начало затирания", "disk", drivePath, "free_space", diskInfo.FreeSize)

	// Создаем временный файл в корне диска
	tempFile := filepath.Join(drivePath, "wipedisk_reserve.tmp")

	// Используем os.OpenFile с эксклюзивными флагами
	file, err := os.OpenFile(tempFile, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания временного файла: %w", err)
	}
	defer func() {
		file.Close()
		os.Remove(tempFile)
	}()

	// Подготовка буфера для записи
	buffer := make([]byte, pfw.config.BufferSize)

	// Если паттерн не указан, генерируем случайные данные
	if pfw.config.Pattern == nil {
		if _, err := rand.Read(buffer); err != nil {
			return nil, fmt.Errorf("ошибка генерации случайных данных: %w", err)
		}
	} else {
		// Копируем паттерн в буфер
		copy(buffer, pfw.config.Pattern)
	}

	var bytesWritten uint64
	var filesCreated int

	// Бесконечный цикл записи до ошибки "Недостаточно места на диске"
	for {
		select {
		case <-ctx.Done():
			// Контекст отменен
			result.Cancelled = true
			result.BytesWritten = bytesWritten
			result.Duration = time.Since(startTime)
			if result.Duration.Seconds() > 0 {
				result.SpeedMBps = float64(bytesWritten) / (1024 * 1024) / result.Duration.Seconds()
			}
			pfw.config.Logger.Log("WARN", "Операция отменена", "disk", drivePath, "bytes_written", bytesWritten)
			return result, ctx.Err()
		default:
		}

		// Запись буфера в файл
		n, err := file.Write(buffer)
		if err != nil {
			// Проверяем на ошибку "Недостаточно места на диске"
			if system.IsDiskFullError(err) {
				pfw.config.Logger.Log("INFO", "Диск заполнен", "disk", drivePath, "bytes_written", bytesWritten)
				break
			}
			pfw.config.Logger.Log("ERROR", "Ошибка записи", "disk", drivePath, "error", err.Error())
			return nil, fmt.Errorf("ошибка записи в файл: %w", err)
		}

		bytesWritten += uint64(n)
		filesCreated++

		// Отправка прогресса
		if pfw.config.Progress != nil {
			progress := ProgressInfo{
				BytesWritten: bytesWritten,
				CurrentFile:  tempFile,
			}

			// Вычисление скорости
			elapsed := time.Since(startTime).Seconds()
			if elapsed > 0 {
				progress.SpeedMBps = float64(bytesWritten) / (1024 * 1024) / elapsed
			}

			// Вычисление процента (если известен общий объем)
			if diskInfo.FreeSize > 0 {
				progress.Percentage = float64(bytesWritten) / float64(diskInfo.FreeSize) * 100
			}

			select {
			case pfw.config.Progress <- progress:
			case <-ctx.Done():
				result.Cancelled = true
				return result, ctx.Err()
			default:
				// Канал прогресса заблокирован, пропускаем
			}
		}

		// Проверка максимальной длительности
		if pfw.config.MaxDuration > 0 && time.Since(startTime) > pfw.config.MaxDuration {
			result.Cancelled = true
			pfw.config.Logger.Log("WARN", "Превышено максимальное время выполнения", "disk", drivePath)
			break
		}
	}

	// ОБЯЗАТЕЛЬНО синхронизируем кэш с физическим носителем
	if err := file.Sync(); err != nil {
		pfw.config.Logger.Log("ERROR", "Ошибка синхронизации файла", "disk", drivePath, "error", err.Error())
		return nil, fmt.Errorf("ошибка синхронизации файла: %w", err)
	}

	// Закрываем файл перед удалением
	if err := file.Close(); err != nil {
		pfw.config.Logger.Log("ERROR", "Ошибка закрытия файла", "disk", drivePath, "error", err.Error())
		return nil, fmt.Errorf("ошибка закрытия файла: %w", err)
	}

	// Удаляем временный файл
	if err := os.Remove(tempFile); err != nil {
		pfw.config.Logger.Log("WARN", "Ошибка удаления временного файла", "disk", drivePath, "error", err.Error())
		// Не считаем это критической ошибкой
	}

	// Формирование результата
	result.Success = true
	result.BytesWritten = bytesWritten
	result.Duration = time.Since(startTime)
	result.FilesCreated = filesCreated

	if result.Duration.Seconds() > 0 {
		result.SpeedMBps = float64(bytesWritten) / (1024 * 1024) / result.Duration.Seconds()
	}

	pfw.config.Logger.Log("INFO", "Затирание завершено", "disk", drivePath,
		"bytes_written", bytesWritten, "duration", result.Duration, "speed_mbps", result.SpeedMBps)

	return result, nil
}

// GetFreeSpacePersistent возвращает свободное место на диске в байтах
func GetFreeSpacePersistent(drivePath string) (uint64, error) {
	diskInfo, err := system.GetDiskInfoForPath(drivePath)
	if err != nil {
		return 0, err
	}
	return diskInfo.FreeSize, nil
}

// IsDiskWritablePersistent проверяет, доступен ли диск для записи
func IsDiskWritablePersistent(drivePath string) bool {
	return system.CheckWriteAccess(drivePath)
}
