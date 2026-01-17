package wipe

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"wipedisk_enterprise/internal/system"
)

// PersistentFileEngine реализует Persistent File Wipe с одним файлом
type PersistentFileEngine struct {
	config *PersistentFileConfig
	mu     sync.Mutex
}

// NewPersistentFileEngine создает новый экземпляр PersistentFileEngine
func NewPersistentFileEngine(config *PersistentFileConfig) *PersistentFileEngine {
	if config.BufferSize <= 0 {
		config.BufferSize = 1024 * 1024 // 1МБ по умолчанию
	}
	return &PersistentFileEngine{config: config}
}

// Wipe выполняет затирание свободного места на указанном диске
func (pfe *PersistentFileEngine) Wipe(ctx context.Context, drivePath string, pattern []byte) (*WipeResult, error) {
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

	pfe.config.Logger.Log("INFO", "Начало Persistent File Wipe", "disk", drivePath, "free_space", diskInfo.FreeSize)

	// Создаем ОДИН временный файл в корне диска
	tempFile := filepath.Join(drivePath, "wipedisk_reserve.tmp")

	// Используем os.OpenFile с эксклюзивными флагами
	file, err := os.OpenFile(tempFile, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0600)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания временного файла: %w", err)
	}

	// Гарантированная очистка ресурсов
	defer func() {
		file.Close()
		os.Remove(tempFile)
		pfe.config.Logger.Log("INFO", "Временный файл удален", "file", tempFile)
	}()

	// Подготовка буфера для записи (1MB)
	buffer := make([]byte, pfe.config.BufferSize)

	// Если паттерн не указан, генерируем случайные данные через crypto/rand
	if pattern == nil {
		if _, err := rand.Read(buffer); err != nil {
			return nil, fmt.Errorf("ошибка генерации случайных данных: %w", err)
		}
	} else {
		// Копируем паттерн в буфер
		copy(buffer, pattern)
	}

	var bytesWritten uint64
	var lastProgressTime time.Time

	// Бесконечный цикл записи до ошибки ERROR_DISK_FULL (код 112)
	for {
		select {
		case <-ctx.Done():
			// Контекст отменен - корректная остановка
			result.Cancelled = true
			result.BytesWritten = bytesWritten
			result.Duration = time.Since(startTime)
			if result.Duration.Seconds() > 0 {
				result.SpeedMBps = float64(bytesWritten) / (1024 * 1024) / result.Duration.Seconds()
			}
			pfe.config.Logger.Log("WARN", "Операция отменена пользователем", "disk", drivePath, "bytes_written", bytesWritten)
			return result, ctx.Err()
		default:
		}

		// Запись буфера в файл
		n, err := file.Write(buffer)
		if err != nil {
			// Проверяем на ошибку ERROR_DISK_FULL (код 112)
			if system.IsDiskFullError(err) {
				pfe.config.Logger.Log("INFO", "Диск заполнен - ERROR_DISK_FULL", "disk", drivePath, "bytes_written", bytesWritten)
				break
			}
			// Другие ошибки записи
			pfe.config.Logger.Log("ERROR", "Ошибка записи в файл", "disk", drivePath, "error", err.Error())
			return nil, fmt.Errorf("ошибка записи в файл: %w", err)
		}

		bytesWritten += uint64(n)

		// Progress reporting каждую секунду
		now := time.Now()
		if now.Sub(lastProgressTime) >= time.Second || pfe.config.Progress != nil {
			lastProgressTime = now

			if pfe.config.Progress != nil {
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
				case pfe.config.Progress <- progress:
				case <-ctx.Done():
					result.Cancelled = true
					return result, ctx.Err()
				default:
					// Канал прогресса заблокирован, пропускаем
				}
			}
		}

		// Проверка максимальной длительности
		if pfe.config.MaxDuration > 0 && time.Since(startTime) > pfe.config.MaxDuration {
			result.Cancelled = true
			pfe.config.Logger.Log("WARN", "Превышено максимальное время выполнения", "disk", drivePath)
			break
		}
	}

	// ОБЯЗАТЕЛЬНО синхронизируем кэш с физическим носителем перед закрытием
	if err := file.Sync(); err != nil {
		pfe.config.Logger.Log("ERROR", "Ошибка синхронизации файла", "disk", drivePath, "error", err.Error())
		return nil, fmt.Errorf("ошибка синхронизации файла: %w", err)
	}

	// Закрываем файл перед удалением (defer все равно сработает)
	if err := file.Close(); err != nil {
		pfe.config.Logger.Log("ERROR", "Ошибка закрытия файла", "disk", drivePath, "error", err.Error())
		return nil, fmt.Errorf("ошибка закрытия файла: %w", err)
	}

	// Удаляем временный файл (defer все равно сработает)
	if err := os.Remove(tempFile); err != nil {
		pfe.config.Logger.Log("WARN", "Ошибка удаления временного файла", "disk", drivePath, "error", err.Error())
		// Не считаем это критической ошибкой, так как основная задача выполнена
	}

	// Формирование результата
	result.Success = true
	result.BytesWritten = bytesWritten
	result.Duration = time.Since(startTime)
	result.FilesCreated = 1 // Один файл

	if result.Duration.Seconds() > 0 {
		result.SpeedMBps = float64(bytesWritten) / (1024 * 1024) / result.Duration.Seconds()
	}

	pfe.config.Logger.Log("INFO", "Persistent File Wipe завершен", "disk", drivePath,
		"bytes_written", bytesWritten, "duration", result.Duration, "speed_mbps", result.SpeedMBps)

	return result, nil
}
