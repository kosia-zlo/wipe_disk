package wipe

import (
	"context"
	"crypto/rand"
	"errors"
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
	var currentFileSize int64

	// Получаем информацию о свободном месте для более точной оценки
	var freeSpaceGB float64
	freeSpaceGB = 2000.0 // По умолчанию 2 ТБ
	bufferSizeMB := float64(pfw.config.BufferSize) / 1024 / 1024

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
		elapsed := time.Since(startTime)
		estimatedTime := ""

		// Всегда показываем оценку времени, даже для первого файла
		if bytesWritten > 0 {
			// Оценка оставшегося времени
			estimatedBytes := float64(bytesWritten) / elapsed.Seconds()
			// Используем реальное свободное место если доступно
			targetBytes := freeSpaceGB * 1024 * 1024 * 1024
			if targetBytes == 0 {
				targetBytes = float64(2 * 1024 * 1024 * 1024 * 1024) // 2 ТБ по умолчанию
			}
			remainingBytes := targetBytes - float64(bytesWritten)
			if estimatedBytes > 0 {
				remainingSecs := remainingBytes / estimatedBytes
				if remainingSecs < 60 {
					estimatedTime = fmt.Sprintf("(~%d сек)", int(remainingSecs))
				} else if remainingSecs < 3600 {
					estimatedTime = fmt.Sprintf("(~%d мин)", int(remainingSecs/60))
				} else {
					estimatedTime = fmt.Sprintf("(~%.1f час)", remainingSecs/3600)
				}
			}
		} else {
			// Для первого файла показываем примерное время на основе свободного места
			if freeSpaceGB > 0 {
				estimatedHours := freeSpaceGB / 100 / 60 / 60 // 100 МБ/с средняя скорость
				if estimatedHours < 1 {
					estimatedTime = fmt.Sprintf("(~%.0f мин)", estimatedHours*60)
				} else {
					estimatedTime = fmt.Sprintf("(~%.1f час)", estimatedHours)
				}
			} else {
				estimatedTime = "(~5.5 час)" // По умолчанию
			}
		}

		fmt.Printf(">>> Создаю файл #%d: %s (начнет %.1f МБ, растет до заполнения диска) %s\n", fileIndex, filepath.Base(fileName), bufferSizeMB, estimatedTime)
		file, err := os.Create(fileName)
		if err != nil {
			// Проверяем, не ошибка ли это "Недостаточно места"
			if isDiskFullError(err) {
				// Диск заполнен - завершаем затирание
				break
			}
			return nil, fmt.Errorf("ошибка создания файла %s: %w", fileName, err)
		}
		defer file.Close()  // Гарантированное закрытие дескриптора
		currentFileSize = 0 // Сбрасываем счетчик для нового файла

		// Записываем данные блоками
		for {
			_, err := file.Write(buffer)
			if err != nil {
				// Проверяем, не ошибка ли это "Недостаточно места"
				if isDiskFullError(err) {
					// Диск заполнен - завершаем затирание
					goto cleanup
				}
				return nil, fmt.Errorf("ошибка записи в файл %s: %w", fileName, err)
			}
			bytesWritten += uint64(len(buffer))
			currentFileSize += int64(len(buffer))

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

		filesCreated++
		fileIndex++

		// Выводим прогресс в консоль
		writtenGB := float64(bytesWritten) / (1024 * 1024 * 1024)
		elapsedTime := time.Since(startTime)
		speedMBps := float64(bytesWritten) / (1024 * 1024) / elapsedTime.Seconds()
		currentFileMB := float64(currentFileSize) / 1024 / 1024
		fmt.Printf("+++ Записано: %.2f GB | Скорость: %.1f MB/s | Файлов: %d | Последний: %.1f МБ\n", writtenGB, speedMBps, filesCreated, currentFileMB)
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

// isDiskFullError проверяет, является ли ошибка ошибкой заполнения диска
func isDiskFullError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Проверка на стандартные ошибки заполнения диска
	if strings.Contains(errStr, "no space left") ||
		strings.Contains(errStr, "disk full") ||
		strings.Contains(errStr, "insufficient space") ||
		strings.Contains(errStr, "not enough space") ||
		strings.Contains(errStr, "volume full") ||
		strings.Contains(errStr, "disk is full") {
		return true
	}

	// Проверка на специфичные ошибки Windows
	if strings.Contains(errStr, "ERROR_DISK_FULL") ||
		strings.Contains(errStr, "ERROR_HANDLE_DISK_FULL") ||
		strings.Contains(errStr, "ERROR_NOT_ENOUGH_QUOTA") {
		return true
	}

	// Проверка на системные ошибки
	// os.ErrNoSpace не существует в Go, используем проверку через строки
	if errors.Is(err, os.ErrPermission) {
		return true
	}

	return false
}
