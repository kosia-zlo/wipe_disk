package wipe

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"time"

	"wipedisk_enterprise/internal/logging"
	"wipedisk_enterprise/internal/system"
)

// WipeMode определяет режим затирания
type WipeMode string

const (
	ModeStandard WipeMode = "standard"
	ModeSDelete  WipeMode = "sdelete"
	ModeCipher   WipeMode = "cipher"
)

// WipeStrategy определяет стратегию затирания
type WipeStrategy interface {
	GetFileSize(diskType string, profile string) uint64
	GetSyncInterval() uint64
	GetMinFreeSpace() uint64
	ShouldCreateMultipleFiles() bool
	GetMaxFiles() int
}

// StandardStrategy реализует безопасную стратегию с множественными файлами
type StandardStrategy struct{}

func (s *StandardStrategy) GetFileSize(diskType string, profile string) uint64 {
	switch profile {
	case "safe":
		if diskType == "SSD" {
			return 2 * 1024 * 1024 * 1024 // 2GB
		}
		return 1 * 1024 * 1024 * 1024 // 1GB
	case "balanced":
		if diskType == "SSD" {
			return 4 * 1024 * 1024 * 1024 // 4GB
		}
		return 2 * 1024 * 1024 * 1024 // 2GB
	case "aggressive":
		if diskType == "SSD" {
			return 8 * 1024 * 1024 * 1024 // 8GB
		}
		return 4 * 1024 * 1024 * 1024 // 4GB
	case "fast":
		if diskType == "SSD" {
			return 16 * 1024 * 1024 * 1024 // 16GB
		}
		return 8 * 1024 * 1024 * 1024 // 8GB
	case "sdelete":
		if diskType == "SSD" {
			return 4 * 1024 * 1024 * 1024 // 4GB
		}
		return 2 * 1024 * 1024 * 1024 // 2GB
	default:
		return 2 * 1024 * 1024 * 1024 // 2GB по умолчанию
	}
}

func (s *StandardStrategy) GetSyncInterval() uint64 {
	return 512 * 1024 * 1024 // 512MB
}

func (s *StandardStrategy) GetMinFreeSpace() uint64 {
	return 100 * 1024 * 1024 // 100MB
}

func (s *StandardStrategy) ShouldCreateMultipleFiles() bool {
	return true
}

func (s *StandardStrategy) GetMaxFiles() int {
	return 1000
}

// SDeleteStrategy реализует агрессивную стратегию как у SDelete
type SDeleteStrategy struct{}

func (s *SDeleteStrategy) GetFileSize(diskType string, profile string) uint64 {
	switch profile {
	case "safe":
		if diskType == "SSD" {
			return 4 * 1024 * 1024 * 1024 // 4GB
		}
		return 4 * 1024 * 1024 * 1024 // 4GB
	case "balanced":
		if diskType == "SSD" {
			return 8 * 1024 * 1024 * 1024 // 8GB
		}
		return 8 * 1024 * 1024 * 1024 // 8GB
	case "aggressive":
		if diskType == "SSD" {
			return 16 * 1024 * 1024 * 1024 // 16GB
		}
		return 16 * 1024 * 1024 * 1024 // 16GB
	case "fast":
		if diskType == "SSD" {
			return 16 * 1024 * 1024 * 1024 // 16GB
		}
		return 16 * 1024 * 1024 * 1024 // 16GB
	case "sdelete":
		if diskType == "SSD" {
			return 16 * 1024 * 1024 * 1024 // 16GB
		}
		return 16 * 1024 * 1024 * 1024 // 16GB
	default:
		return 8 * 1024 * 1024 * 1024 // 8GB по умолчанию
	}
}

func (s *SDeleteStrategy) GetSyncInterval() uint64 {
	return 1024 * 1024 * 1024 // 1GB
}

func (s *SDeleteStrategy) GetMinFreeSpace() uint64 {
	return 50 * 1024 * 1024 // 50MB
}

func (s *SDeleteStrategy) ShouldCreateMultipleFiles() bool {
	return true
}

func (s *SDeleteStrategy) GetMaxFiles() int {
	return 100 // Меньше файлов, чем в standard
}

// CipherStrategy реализует стратегию совместимую с cipher /w
type CipherStrategy struct{}

func (s *CipherStrategy) GetFileSize(diskType string, profile string) uint64 {
	if diskType == "SSD" {
		return 32 * 1024 * 1024 * 1024 // 32GB
	}
	return 16 * 1024 * 1024 * 1024 // 16GB
}

func (s *CipherStrategy) GetSyncInterval() uint64 {
	return 2048 * 1024 * 1024 // 2GB
}

func (s *CipherStrategy) GetMinFreeSpace() uint64 {
	return 10 * 1024 * 1024 // 10MB
}

func (s *CipherStrategy) ShouldCreateMultipleFiles() bool {
	return true
}

func (s *CipherStrategy) GetMaxFiles() int {
	return 50
}

// GetStrategy возвращает стратегию по режиму
func GetStrategy(mode WipeMode) WipeStrategy {
	switch mode {
	case ModeStandard:
		return &StandardStrategy{}
	case ModeSDelete:
		return &SDeleteStrategy{}
	case ModeCipher:
		return &CipherStrategy{}
	default:
		return &StandardStrategy{}
	}
}

// CipherPass определяет тип прохода для cipher режима
type CipherPass int

const (
	CipherPassZero CipherPass = iota
	CipherPassFF
	CipherPassRandom
)

func (p CipherPass) String() string {
	switch p {
	case CipherPassZero:
		return "zero"
	case CipherPassFF:
		return "0xFF"
	case CipherPassRandom:
		return "random"
	default:
		return "unknown"
	}
}

// CipherPattern генерирует паттерн для cipher прохода
func CipherPattern(pass CipherPass, size int) ([]byte, error) {
	buf := make([]byte, size)

	switch pass {
	case CipherPassZero:
	case CipherPassFF:
		for i := range buf {
			buf[i] = 0xFF
		}
	case CipherPassRandom:
		if _, err := rand.Read(buf); err != nil {
			return nil, fmt.Errorf("ошибка генерации случайных данных: %w", err)
		}
	default:
		return nil, fmt.Errorf("неизвестный проход cipher: %d", pass)
	}

	return buf, nil
}

// ExecuteWipeWithStrategy выполняет затирание с использованием стратегии
func ExecuteWipeWithStrategy(ctx context.Context, disk system.DiskInfo, cfg *WipeConfig, logger *logging.EnterpriseLogger, mode WipeMode, profile string) (*WipeOperation, error) {
	strategy := GetStrategy(mode)

	op := &WipeOperation{
		ID:        fmt.Sprintf("wipe_%d", time.Now().UnixNano()),
		Disk:      disk.Letter,
		Method:    string(mode),
		Passes:    cfg.Passes,
		ChunkSize: int64(strategy.GetFileSize(disk.Type, profile)),
		Status:    "RUNNING",
		StartTime: time.Now(),
	}

	logger.Log("INFO", "Запуск затирания", "disk", disk.Letter, "mode", mode, "profile", profile, "strategy", fmt.Sprintf("%T", strategy))

	passes := cfg.Passes
	if mode == ModeCipher {
		passes = 3
	}

	for pass := 0; pass < passes; pass++ {
		// Проверка контекста
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				op.Status = "PARTIAL"
				op.Warning = "Операция прервана по таймауту"
			} else {
				op.Status = "CANCELLED"
				op.Warning = "Операция отменена пользователем"
			}
			return op, fmt.Errorf("операция прервана")
		default:
		}

		var err error
		if mode == ModeCipher {
			cipherPass := CipherPass(pass)
			err = executeCipherPass(ctx, disk, cfg, logger, strategy, cipherPass, profile)
			logger.Log("INFO", "Cipher проход завершен", "disk", disk.Letter, "pass", cipherPass.String(), "error", err)
		} else {
			err = executeStandardPass(ctx, disk, cfg, logger, strategy, profile, pass)
			logger.Log("INFO", "Проход завершен", "disk", disk.Letter, "pass", pass+1, "total", passes, "error", err)
		}

		if err != nil {
			if ctx.Err() != nil {
				if ctx.Err() == context.DeadlineExceeded {
					op.Status = "PARTIAL"
					op.Warning = "Операция прервана по таймауту"
				} else {
					op.Status = "CANCELLED"
					op.Warning = "Операция отменена пользователем"
				}
			} else {
				op.Status = "FAILED"
				op.Error = err.Error()
			}
			return op, err
		}
	}

	// Успешное завершение
	now := time.Now()
	op.EndTime = &now
	op.Status = "COMPLETED"
	op.BytesWiped = disk.FreeSize
	if op.EndTime.Sub(op.StartTime).Seconds() > 0 {
		op.SpeedMBps = float64(op.BytesWiped) / (1024 * 1024) / op.EndTime.Sub(op.StartTime).Seconds()
	}

	logger.Log("INFO", "Затирание завершено", "disk", disk.Letter, "bytes", op.BytesWiped, "speed", op.SpeedMBps)
	return op, nil
}

// executeStandardPass выполняет стандартный проход затирания
func executeStandardPass(ctx context.Context, disk system.DiskInfo, cfg *WipeConfig, logger *logging.EnterpriseLogger, strategy WipeStrategy, profile string, passNum int) error {
	freeSpace := disk.FreeSize
	minFreeSpace := strategy.GetMinFreeSpace()
	maxFiles := strategy.GetMaxFiles()
	fileSize := strategy.GetFileSize(disk.Type, profile)
	syncInterval := strategy.GetSyncInterval()

	fileIndex := 0

	for freeSpace > minFreeSpace && fileIndex < maxFiles {
		// Проверка контекста
		select {
		case <-ctx.Done():
			return fmt.Errorf("операция отменена")
		default:
		}

		// Определяем размер файла
		currentFileSize := fileSize
		if freeSpace < currentFileSize {
			currentFileSize = freeSpace - minFreeSpace
			if currentFileSize < 64*1024*1024 { // Минимальный размер файла 64MB
				break
			}
		}

		// Создаем и заполняем файл
		filename := fmt.Sprintf("%swipe_%03d.tmp", disk.Letter, fileIndex)
		err := createLargeFile(ctx, filename, currentFileSize, cfg.MaxSpeedMBps, syncInterval, logger)
		if err != nil {
			return fmt.Errorf("ошибка создания файла %s: %w", filename, err)
		}

		// Удаляем файл
		if err := os.Remove(filename); err != nil {
			logger.Log("WARN", "Ошибка удаления файла", "file", filename, "error", err.Error())
		}

		freeSpace -= currentFileSize
		fileIndex++
	}

	return nil
}

// executeCipherPass выполняет проход cipher с указанным паттерном
func executeCipherPass(ctx context.Context, disk system.DiskInfo, cfg *WipeConfig, logger *logging.EnterpriseLogger, strategy WipeStrategy, pass CipherPass, profile string) error {
	freeSpace := disk.FreeSize
	minFreeSpace := strategy.GetMinFreeSpace()
	maxFiles := strategy.GetMaxFiles()
	fileSize := strategy.GetFileSize(disk.Type, profile)
	syncInterval := strategy.GetSyncInterval()

	fileIndex := 0

	for freeSpace > minFreeSpace && fileIndex < maxFiles {
		// Проверка контекста
		select {
		case <-ctx.Done():
			return fmt.Errorf("операция отменена")
		default:
		}

		// Определяем размер файла
		currentFileSize := fileSize
		if freeSpace < currentFileSize {
			currentFileSize = freeSpace - minFreeSpace
			if currentFileSize < 64*1024*1024 { // Минимальный размер файла 64MB
				break
			}
		}

		// Создаем и заполняем файл с cipher паттерном
		filename := fmt.Sprintf("%scipher_%03d_%s.tmp", disk.Letter, fileIndex, pass.String())
		err := createCipherFile(ctx, filename, currentFileSize, cfg.MaxSpeedMBps, syncInterval, pass, logger)
		if err != nil {
			return fmt.Errorf("ошибка создания cipher файла %s: %w", filename, err)
		}

		// Удаляем файл
		if err := os.Remove(filename); err != nil {
			logger.Log("WARN", "Ошибка удаления cipher файла", "file", filename, "error", err.Error())
		}

		freeSpace -= currentFileSize
		fileIndex++
	}

	return nil
}

// createLargeFile создает большой файл с последовательной записью
func createLargeFile(ctx context.Context, filename string, fileSize uint64, maxSpeedMBps float64, syncInterval uint64, logger *logging.EnterpriseLogger) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	throttledWriter := NewThrottledWriter(file, maxSpeedMBps)

	// Используем большой буфер для последовательной записи
	chunkSize := 16 * 1024 * 1024 // 16MB chunks
	buf := make([]byte, chunkSize)

	// Заполняем буфер случайными данными один раз
	if _, err := rand.Read(buf); err != nil {
		return fmt.Errorf("ошибка генерации данных: %w", err)
	}

	var written uint64
	lastSync := uint64(0)

	for written < fileSize {
		// Проверка контекста
		select {
		case <-ctx.Done():
			return fmt.Errorf("операция отменена")
		default:
		}

		remaining := fileSize - written
		toWrite := uint64(chunkSize)
		if remaining < toWrite {
			toWrite = remaining
		}

		// Записываем данные
		off := 0
		for off < int(toWrite) {
			n, err := throttledWriter.Write(buf[off:int(toWrite)])
			if n > 0 {
				off += n
				written += uint64(n)
			}
			if err != nil {
				return fmt.Errorf("ошибка записи: %w", err)
			}
			if n == 0 {
				return fmt.Errorf("запись вернула 0 байт")
			}
		}

		// Периодический sync
		if syncInterval > 0 && written-lastSync >= syncInterval {
			if err := file.Sync(); err != nil {
				return fmt.Errorf("ошибка синхронизации: %w", err)
			}
			lastSync = written
		}
	}

	// Финальный sync
	if err := file.Sync(); err != nil {
		return fmt.Errorf("ошибка финальной синхронизации: %w", err)
	}

	return nil
}

// createCipherFile создает файл с cipher паттерном
func createCipherFile(ctx context.Context, filename string, fileSize uint64, maxSpeedMBps float64, syncInterval uint64, pass CipherPass, logger *logging.EnterpriseLogger) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	throttledWriter := NewThrottledWriter(file, maxSpeedMBps)

	// Используем большой буфер для последовательной записи
	chunkSize := 16 * 1024 * 1024 // 16MB chunks

	var written uint64
	lastSync := uint64(0)

	for written < fileSize {
		// Проверка контекста
		select {
		case <-ctx.Done():
			return fmt.Errorf("операция отменена")
		default:
		}

		remaining := fileSize - written
		toWrite := uint64(chunkSize)
		if remaining < toWrite {
			toWrite = remaining
		}

		// Генерируем паттерн для этого чанка
		pattern, err := CipherPattern(pass, int(toWrite))
		if err != nil {
			return fmt.Errorf("ошибка генерации паттерна: %w", err)
		}

		// Записываем данные
		off := 0
		for off < int(toWrite) {
			n, err := throttledWriter.Write(pattern[off:int(toWrite)])
			if n > 0 {
				off += n
				written += uint64(n)
			}
			if err != nil {
				return fmt.Errorf("ошибка записи: %w", err)
			}
			if n == 0 {
				return fmt.Errorf("запись вернула 0 байт")
			}
		}

		// Периодический sync
		if syncInterval > 0 && written-lastSync >= syncInterval {
			if err := file.Sync(); err != nil {
				return fmt.Errorf("ошибка синхронизации: %w", err)
			}
			lastSync = written
		}
	}

	// Финальный sync
	if err := file.Sync(); err != nil {
		return fmt.Errorf("ошибка финальной синхронизации: %w", err)
	}

	return nil
}

// WipeConfig конфигурация для затирания
type WipeConfig struct {
	Passes       int
	MaxSpeedMBps float64
	MaxDuration  time.Duration
}
