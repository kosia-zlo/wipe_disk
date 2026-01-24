package wipe

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"wipedisk_enterprise/internal/logging"
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

	// Создаем временную скрытую директорию в корне диска
	tempDir := filepath.Join(drivePath, ".wipedisk_tmp")
	var err error
	err = os.MkdirAll(tempDir, 0700)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания временной директории: %w", err)
	}
	defer func() {
		// Удаляем временную директорию после завершения
		os.RemoveAll(tempDir)
	}()

	// Подготовка буфера для записи (1 МБ)
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
	fileIndex := 1

	// Бесконечный цикл создания файлов до ошибки "Недостаточно места на диске"
	for {
		// Проверка контекста
		select {
		case <-ctx.Done():
			result.Cancelled = true
			return result, fmt.Errorf("операция отменена")
		default:
		}

		// Создаем новый файл
		fileName := filepath.Join(tempDir, fmt.Sprintf("wipe_data_%d.bin", fileIndex))
		fmt.Printf(">>> Создаю файл: %s\n", fileName)
		file, err := os.Create(fileName)
		if err != nil {
			// Проверяем, не ошибка ли это "Недостаточно места"
			if strings.Contains(err.Error(), "no space left") || strings.Contains(err.Error(), "disk full") {
				// Диск заполнен - завершаем затирание
				break
			}
			return nil, fmt.Errorf("ошибка создания файла %s: %w", fileName, err)
		}

		// Записываем данные блоками
		for {
			_, err := file.Write(buffer)
			if err != nil {
				file.Close()
				// Проверяем, не ошибка ли это "Недостаточно места"
				if strings.Contains(err.Error(), "no space left") || strings.Contains(err.Error(), "disk full") {
					// Диск заполнен - завершаем затирание
					goto cleanup
				}
				return nil, fmt.Errorf("ошибка записи в файл %s: %w", fileName, err)
			}
			bytesWritten += uint64(len(buffer))

			// Отправляем прогресс каждые 100 МБ
			if pfw.config.Progress != nil && bytesWritten%(100*1024*1024) == 0 {
				progress := ProgressInfo{
					BytesWritten: bytesWritten,
					SpeedMBps:    float64(bytesWritten) / time.Since(startTime).Seconds() / (1024 * 1024),
					Percentage:   0, // Не можем рассчитать без info о свободном месте
					CurrentFile:  fileName,
					StartTime:    startTime,
				}
				pfw.config.Progress <- progress
			}
		}

		file.Close()
		filesCreated++
		fileIndex++

		// Выводим прогресс в консоль
		writtenGB := float64(bytesWritten) / (1024 * 1024 * 1024)
		fmt.Printf("+++ Записано: %.2f GB\n", writtenGB)
	}

cleanup:

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
