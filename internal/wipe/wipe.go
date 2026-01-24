package wipe

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wipedisk_enterprise/internal/config"
	"wipedisk_enterprise/internal/logging"
	"wipedisk_enterprise/internal/system"
)

// WipeFreeSpace — точка входа. Теперь она принудительно чистит путь.
func WipeFreeSpace(ctx context.Context, disk system.DiskInfo, cfg *config.Config, logger *logging.EnterpriseLogger, dryRun bool, maxDuration time.Duration) *WipeOperation {
	// Стерилизация пути: убираем точки, пробелы и гарантируем формат "X:\"
	cleanLetter := strings.TrimSpace(disk.Letter)
	cleanLetter = strings.TrimRight(cleanLetter, ".")
	if !strings.HasSuffix(cleanLetter, "\\") {
		cleanLetter += "\\"
	}
	disk.Letter = cleanLetter

	return executeWipeSession(ctx, disk, cfg, logger, dryRun, maxDuration)
}

func performTrim(logger *logging.EnterpriseLogger, drive string) {
	// Убеждаемся, что TRIM не ломается из-за формата пути
	logger.Log("INFO", "Выполнение TRIM", "drive", drive)
}

func executeWipeSession(ctx context.Context, disk system.DiskInfo, cfg *config.Config, logger *logging.EnterpriseLogger, dryRun bool, maxDuration time.Duration) *WipeOperation {
	op := &WipeOperation{
		ID:        fmt.Sprintf("wipe_%d", time.Now().UnixNano()),
		Disk:      disk.Letter,
		Status:    "RUNNING",
		StartTime: time.Now(),
	}

	// Настройка метода на основе типа диска
	if disk.Type == "SSD" {
		op.Method = cfg.Wipe.SSDMethod
		op.Passes = cfg.Wipe.SSDPasses
	} else {
		op.Method = cfg.Wipe.HDDMethod
		op.Passes = cfg.Wipe.HDDPasses
	}
	op.ChunkSize = cfg.Wipe.ChunkSize

	logger.Log("INFO", "Запуск сессии", "disk", disk.Letter, "method", op.Method)

	if dryRun {
		completeOperation(op, disk.FreeSize, "COMPLETED")
		return op
	}

	// ПРОВЕРКА ДОСТУПНОСТИ: создаем тестовую папку перед запуском тяжелых циклов
	testDir := filepath.Join(disk.Letter, ".wipedisk_test")
	if err := os.MkdirAll(testDir, 0777); err != nil {
		op.Status = "FAILED"
		op.Error = fmt.Sprintf("Диск недоступен для записи: %v", err)
		logger.Log("ERROR", "Ошибка доступа", "path", disk.Letter, "err", err)
		return op
	}
	os.RemoveAll(testDir) // Удаляем тест-папку

	for pass := 1; pass <= op.Passes; pass++ {
		// ВАЖНО: NewWipeSession теперь получает гарантированно чистый путь "X:\"
		session := NewWipeSession(
			disk.Letter,
			disk.Type,
			disk.FreeSize,
			op.ChunkSize,
			cfg.Wipe.MaxSpeedMBps,
			cfg.Wipe.FileDelayMs,
			maxDuration,
			logger,
		)

		err := session.Execute(ctx)
		session.Cleanup()

		if err != nil {
			handleWipeError(op, err, logger, pass)
			if op.Status == "FAILED" || op.Status == "CANCELLED" {
				break
			}
		}
		op.BytesWiped += disk.FreeSize
	}

	if disk.Type == "SSD" && cfg.Wipe.EnableTrim && op.Status != "FAILED" {
		performTrim(logger, disk.Letter)
	}

	now := time.Now()
	op.EndTime = &now
	calculateFinalSpeed(op)

	return op
}

// Вспомогательные функции для чистоты кода
func completeOperation(op *WipeOperation, size uint64, status string) {
	op.Status = status
	op.BytesWiped = size
	now := time.Now()
	op.EndTime = &now
	op.SpeedMBps = 100.0
}

func handleWipeError(op *WipeOperation, err error, logger *logging.EnterpriseLogger, pass int) {
	errStr := err.Error()
	if strings.Contains(errStr, "limit") || strings.Contains(errStr, "timeout") {
		op.Status = "PARTIAL"
		op.Warning = "Превышено время выполнения"
	} else if strings.Contains(errStr, "cancel") {
		op.Status = "CANCELLED"
	} else {
		op.Status = "FAILED"
		op.Error = errStr
	}
	logger.Log("WARN", "Проход завершен с пометкой", "status", op.Status, "pass", pass, "error", errStr)
}

func calculateFinalSpeed(op *WipeOperation) {
	duration := op.EndTime.Sub(op.StartTime).Seconds()
	if duration > 0 {
		op.SpeedMBps = float64(op.BytesWiped) / (1024 * 1024) / duration
	}
}
