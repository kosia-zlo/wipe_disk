package wipe

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"wipedisk_enterprise/internal/logging"
	"wipedisk_enterprise/internal/system"
)

// WipeSession управляет сессией затирания для одного диска
type WipeSession struct {
	Disk         string
	DiskType     string
	FreeSpace    uint64
	ChunkSize    int64
	MaxSpeedMBps float64
	FileDelayMs  int
	MaxDuration  time.Duration
	StartTime    time.Time
	CreatedFiles []string
	BytesWritten uint64
	InitialFree  uint64
	Logger       *logging.EnterpriseLogger
}

// NewWipeSession создаёт новую сессию затирания
func NewWipeSession(disk, diskType string, freeSpace uint64, chunkSize int64, maxSpeedMBps float64, fileDelayMs int, maxDuration time.Duration, logger *logging.EnterpriseLogger) *WipeSession {
	return &WipeSession{
		Disk:         disk,
		DiskType:     diskType,
		FreeSpace:    freeSpace,
		ChunkSize:    chunkSize,
		MaxSpeedMBps: maxSpeedMBps,
		FileDelayMs:  fileDelayMs,
		MaxDuration:  maxDuration,
		StartTime:    time.Now(),
		CreatedFiles: make([]string, 0),
		InitialFree:  freeSpace,
		Logger:       logger,
	}
}

// getAdaptiveChunkSize определяет адаптивный размер чанка
func getAdaptiveChunkSize(diskType string) int64 {
	switch diskType {
	case "HDD":
		return 2 * 1024 * 1024 // 2MB для HDD
	case "SSD":
		return 16 * 1024 * 1024 // 16MB для SSD
	default:
		return 4 * 1024 * 1024 // 4MB по умолчанию
	}
}

// createWipeFile создает временный файл для затирания
func (ws *WipeSession) createWipeFile(fileIndex int, fileSize uint64) error {
	filename := fmt.Sprintf("%swipe_%03d.tmp", ws.Disk, fileIndex)

	// Progress output
	ws.printProgress(filename, fileSize)

	// Timeout check
	if ws.MaxDuration > 0 && time.Since(ws.StartTime) > ws.MaxDuration {
		return fmt.Errorf("operation time limit reached")
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			ws.Logger.Log("WARN", "Error closing file", "file", filename, "error", closeErr.Error())
		}
	}()

	throttledWriter := NewThrottledWriter(file, ws.MaxSpeedMBps)
	defer func() {
		if closeErr := throttledWriter.Close(); closeErr != nil {
			ws.Logger.Log("WARN", "Error closing throttled writer", "file", filename, "error", closeErr.Error())
		}
	}()

	chunkSize := int(getAdaptiveChunkSize(ws.DiskType))
	if chunkSize <= 0 {
		chunkSize = 1024 * 1024 // 1MB fallback
	}

	// Используем buffer pool для оптимизации памяти
	buf := GetBuffer(chunkSize)
	defer PutBuffer(buf)

	var written uint64
	for written < fileSize {
		// Проверка на переполнение
		if written > uint64(^uint64(0)-uint64(chunkSize)) {
			return fmt.Errorf("file size overflow")
		}

		remaining := fileSize - written
		toWrite := chunkSize
		if remaining < uint64(toWrite) {
			toWrite = int(remaining)
		}

		if toWrite <= 0 {
			break
		}

		b := buf[:toWrite]
		if err := FillRandom(b); err != nil {
			ws.Logger.Log("ERROR", "Failed to generate random data", "error", err.Error())
			return fmt.Errorf("data generation error: %w", err)
		}

		off := 0
		for off < toWrite {
			n, err := throttledWriter.Write(b[off:])
			if n > 0 {
				off += n
				written += uint64(n)
				ws.BytesWritten += uint64(n)
			}
			if err != nil {
				ws.Logger.Log("ERROR", "Write error", "file", filename, "error", err.Error())
				return fmt.Errorf("file write error: %w", err)
			}
			if n == 0 {
				ws.Logger.Log("ERROR", "Zero bytes written without error", "file", filename)
				return fmt.Errorf("write returned 0 bytes without error")
			}
		}
	}

	if err := throttledWriter.Sync(); err != nil {
		ws.Logger.Log("ERROR", "Sync error", "file", filename, "error", err.Error())
		return fmt.Errorf("sync error: %w", err)
	}

	ws.CreatedFiles = append(ws.CreatedFiles, filename)

	// Защита от отрицательного значения
	if ws.FreeSpace >= fileSize {
		ws.FreeSpace -= fileSize
	} else {
		ws.FreeSpace = 0
	}

	return nil
}

// printProgress выводит информацию о прогрессе
func (ws *WipeSession) printProgress(currentFile string, currentFileSize uint64) {
	percent := float64(ws.BytesWritten) / float64(ws.InitialFree) * 100
	elapsed := time.Since(ws.StartTime)

	var eta string
	if ws.MaxSpeedMBps > 0 && percent < 100 {
		remainingBytes := float64(ws.InitialFree - ws.BytesWritten)
		remainingSeconds := remainingBytes / (1024 * 1024) / ws.MaxSpeedMBps
		eta = fmt.Sprintf("ETA: %02d:%02d:%02d",
			int(remainingSeconds)/3600,
			int(remainingSeconds)%3600/60,
			int(remainingSeconds)%60)
	} else if percent >= 100 {
		eta = "Completed"
	} else {
		eta = "Calculating..."
	}

	remainingGB := float64(ws.FreeSpace) / (1024 * 1024 * 1024)

	if currentFile != "" {
		currentSizeGB := float64(currentFileSize) / (1024 * 1024 * 1024)
		fmt.Printf("\r[Disk %s] Creating %s (Size: %.1f GB) | Progress: %.1f%% | Remaining free: %.1f GB | Elapsed: %02d:%02d:%02d | %s",
			ws.Disk, currentFile, currentSizeGB, percent, remainingGB,
			int(elapsed.Hours()), int(elapsed.Minutes())%60, int(elapsed.Seconds())%60,
			eta)
	} else {
		fmt.Printf("\r[Disk %s] Progress: %.1f%% | Remaining free: %.1f GB | Elapsed: %02d:%02d:%02d | %s",
			ws.Disk, percent, remainingGB,
			int(elapsed.Hours()), int(elapsed.Minutes())%60, int(elapsed.Seconds())%60,
			eta)
	}

	if ws.Logger != nil {
		ws.Logger.Log("INFO", fmt.Sprintf("Wipe progress: disk=%s, progress=%.1f%%, remaining=%.1fGB, files=%d",
			ws.Disk, percent, remainingGB, len(ws.CreatedFiles)))
	}
}

func (ws *WipeSession) Execute(ctx context.Context) error {
	const minThreshold = 1 * 1024 * 1024 * 1024 // 1GB
	fileIndex := 1
	const maxFiles = 1000       // Защита от бесконечного цикла
	const maxIterations = 10000 // Дополнительная защита

	previousFreeSpace := ws.FreeSpace
	stuckCounter := 0

	for ws.FreeSpace > minThreshold && fileIndex <= maxFiles {
		// Проверка контекста
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				ws.Logger.Log("WARN", "Сессия прервана по таймауту", "disk", ws.Disk)
				return fmt.Errorf("достигнут лимит времени операции")
			} else {
				ws.Logger.Log("WARN", "Сессия отменена пользователем", "disk", ws.Disk)
				return fmt.Errorf("операция отменена")
			}
		default:
		}

		// Проверка таймаута
		if ws.MaxDuration > 0 && time.Since(ws.StartTime) > ws.MaxDuration {
			ws.Logger.Log("INFO", "Достигнут лимит времени операции", "disk", ws.Disk)
			return fmt.Errorf("достигнут лимит времени операции")
		}

		// Защита от зависания - если свободное место не уменьшается
		if ws.FreeSpace == previousFreeSpace {
			stuckCounter++
			if stuckCounter > 3 {
				ws.Logger.Log("WARN", "Свободное место не уменьшается, возможен бесконечный цикл", "disk", ws.Disk)
				break
			}
		} else {
			stuckCounter = 0
			previousFreeSpace = ws.FreeSpace
		}

		// Рассчитываем размер файла
		maxFileSize := (ws.FreeSpace / 10) / 2
		if maxFileSize > 50*1024*1024*1024 {
			maxFileSize = 50 * 1024 * 1024 * 1024 // Ограничиваем 50GB
		}

		minFileSize := uint64(1024 * 1024 * 1024) // 1GB минимум
		if maxFileSize < minFileSize {
			maxFileSize = minFileSize
		}

		var fileSize uint64
		if maxFileSize == minFileSize {
			fileSize = minFileSize
		} else {
			fileSize = minFileSize + uint64(rand.Int63n(int64(maxFileSize-minFileSize)))
		}
		if fileSize > ws.FreeSpace {
			fileSize = ws.FreeSpace
		}

		// Создаём файл
		err := ws.createWipeFile(fileIndex, fileSize)
		if err != nil {
			// Проверка на специфические ошибки Windows
			if system.IsWindowsError(err, system.ERROR_DISK_FULL) {
				ws.Logger.Log("INFO", "Свободное место исчерпано", "disk", ws.Disk)
				return nil // Нормальное завершение
			}

			if system.IsWindowsError(err, system.ERROR_NOT_READY) {
				ws.Logger.Log("WARN", "Диск недоступен", "disk", ws.Disk, "error", err.Error())
				return fmt.Errorf("диск недоступен: %w", err)
			}

			return fmt.Errorf("ошибка создания файла: %w", err)
		}

		fileIndex++

		// Пауза между файлами для снижения нагрузки
		if ws.FileDelayMs > 0 {
			// Дополнительная проверка контекста во время паузы
			select {
			case <-ctx.Done():
				if ctx.Err() == context.DeadlineExceeded {
					ws.Logger.Log("WARN", "Сессия прервана по таймауту во время паузы", "disk", ws.Disk)
					return fmt.Errorf("достигнут лимит времени операции")
				} else {
					ws.Logger.Log("WARN", "Сессия отменена пользователем во время паузы", "disk", ws.Disk)
					return fmt.Errorf("операция отменена")
				}
			case <-time.After(time.Duration(ws.FileDelayMs) * time.Millisecond):
				// Продолжаем работу
			}
		}

		// Принудительный выход если слишком много итераций
		if fileIndex > maxIterations {
			ws.Logger.Log("WARN", "Достигнут лимит итераций", "disk", ws.Disk, "iterations", fileIndex)
			break
		}
	}

	// Финальный прогресс
	ws.printProgress("", 0)
	fmt.Println() // Новая строка после завершения

	return nil
}

// Cleanup удаляет все созданные временные файлы
func (ws *WipeSession) Cleanup() {
	for _, filename := range ws.CreatedFiles {
		if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
			ws.Logger.Log("WARN", "Ошибка удаления временного файла", "file", filename, "error", err.Error())
		}
	}
	ws.CreatedFiles = ws.CreatedFiles[:0] // Очищаем слайс
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "достигнут лимит времени")
}

// GetDefaultSystemDiskPolicy возвращает политику по умолчанию
func GetDefaultSystemDiskPolicy() *SystemDiskPolicy {
	return &SystemDiskPolicy{
		AllowedPaths: []string{
			"%WINDIR%\\Temp",
			"%TEMP%",
			"%USERPROFILE%\\AppData\\Local\\Temp",
		},
		MaxTempSizeGB:   2,     // Максимум 2GB временных файлов
		MaxBufferMB:     256,   // 256MB буфер
		MaxConcurrentIO: 2,     // Максимум 2 одновременных операции
		TimeoutMinutes:  30,    // 30 минут максимум
		ForceWipeSSD:    false, // Не затирать SSD без флага
	}
}

// PrepareSystemDiskWipe готовит системный диск к безопасному затиранию
func PrepareSystemDiskWipe(disk string, allowSystemDisk bool, logger *logging.EnterpriseLogger) (*SystemDiskPolicy, error) {
	// Проверяем, что это системный диск
	if !strings.EqualFold(disk, "C:") && !strings.EqualFold(disk, "C:\\") {
		return nil, nil // Не системный диск, политика не нужна
	}

	if !allowSystemDisk {
		return nil, fmt.Errorf("затирание системного диска %s запрещено. Используйте --allow-system-disk для принудительного разрешения", disk)
	}

	policy := GetDefaultSystemDiskPolicy()

	logger.Log("WARN", "Подготовка к затиранию системного диска",
		"disk", disk,
		"max_temp_size_gb", policy.MaxTempSizeGB,
		"timeout_minutes", policy.TimeoutMinutes,
		"force_wipe_ssd", policy.ForceWipeSSD)

	// Проверяем тип диска
	diskInfo, err := system.GetDiskInfo(false)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения информации о диске: %w", err)
	}

	for _, info := range diskInfo {
		if strings.EqualFold(info.Letter, disk) {
			if info.Type == "SSD" && !policy.ForceWipeSSD {
				logger.Log("INFO", "Обнаружен SSD системный диск, рекомендуется использовать cipher /w", "disk", disk)
				return nil, fmt.Errorf("затирание SSD системного диска запрещено. Используйте --force-wipe-ssd или cipher движок")
			}
			break
		}
	}

	// Проверяем доступные пути
	for _, path := range policy.AllowedPaths {
		expandedPath := os.ExpandEnv(path)
		if err := os.MkdirAll(expandedPath, 0755); err != nil {
			logger.Log("WARN", "Не удалось создать путь", "path", expandedPath, "error", err)
		} else {
			logger.Log("INFO", "Путь для временных файлов доступен", "path", expandedPath)
		}
	}

	return policy, nil
}
