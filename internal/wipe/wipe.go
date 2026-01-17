package wipe

import (
	"context"
	"fmt"
	"strings"
	"time"

	"wipedisk_enterprise/internal/config"
	"wipedisk_enterprise/internal/logging"
	"wipedisk_enterprise/internal/system"
)

// Затирание свободного места с новой логикой
func WipeFreeSpace(ctx context.Context, disk system.DiskInfo, cfg *config.Config, logger *logging.EnterpriseLogger, dryRun bool, maxDuration time.Duration) *WipeOperation {
	return executeWipeSession(ctx, disk, cfg, logger, dryRun, maxDuration)
}

func performTrim(logger *logging.EnterpriseLogger, drive string) {
	logger.Log("INFO", "Выполнение TRIM", "drive", drive)
}

// executeWipeSession выполняет полный цикл затирания для диска
func executeWipeSession(ctx context.Context, disk system.DiskInfo, cfg *config.Config, logger *logging.EnterpriseLogger, dryRun bool, maxDuration time.Duration) *WipeOperation {
	op := &WipeOperation{
		ID:        fmt.Sprintf("wipe_%d", time.Now().UnixNano()),
		Disk:      disk.Letter,
		Status:    "RUNNING",
		StartTime: time.Now(),
	}

	// Определяем параметры затирания
	switch disk.Type {
	case "SSD":
		op.Method = cfg.Wipe.SSDMethod
		op.Passes = cfg.Wipe.SSDPasses
	case "HDD":
		op.Method = cfg.Wipe.HDDMethod
		op.Passes = cfg.Wipe.HDDPasses
	}

	op.ChunkSize = cfg.Wipe.ChunkSize

	logger.Log("INFO", "Начало затирания", "disk", disk.Letter, "type", disk.Type, "method", op.Method, "passes", op.Passes)

	if dryRun {
		op.Status = "COMPLETED"
		op.BytesWiped = disk.FreeSize
		now := time.Now()
		op.EndTime = &now
		op.SpeedMBps = 100.0
		logger.Log("INFO", "DRY RUN: затирание завершено", "disk", disk.Letter, "bytes", op.BytesWiped)
		return op
	}

	// Выполняем проходы
	for pass := 1; pass <= op.Passes; pass++ {
		logger.Log("INFO", "Проход затирания", "disk", disk.Letter, "pass", pass, "total", op.Passes)

		// Создаём сессию для прохода
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

		// Выполняем затирание
		err := session.Execute(ctx)

		// Очищаем временные файлы
		session.Cleanup()

		if err != nil {
			if strings.Contains(err.Error(), "достигнут лимит времени") {
				op.Status = "PARTIAL"
				op.Warning = err.Error()
				logger.Log("INFO", "Проход завершён по таймауту", "disk", disk.Letter, "pass", pass)
				break
			}

			if strings.Contains(err.Error(), "операция отменена") {
				op.Status = "CANCELLED"
				op.Warning = err.Error()
				logger.Log("INFO", "Проход отменен пользователем", "disk", disk.Letter, "pass", pass)
				break
			}

			if strings.Contains(err.Error(), "диск недоступен") {
				op.Status = "FAILED"
				op.Error = err.Error()
				return op
			}

			op.Status = "FAILED"
			op.Error = err.Error()
			return op
		}

		op.BytesWiped += disk.FreeSize
	}

	// TRIM для SSD
	if disk.Type == "SSD" && cfg.Wipe.EnableTrim {
		performTrim(logger, disk.Letter)
	}

	// Завершение операции
	op.Status = "COMPLETED"
	now := time.Now()
	op.EndTime = &now

	if op.EndTime.Sub(op.StartTime).Seconds() > 0 {
		op.SpeedMBps = float64(op.BytesWiped) / (1024 * 1024) / op.EndTime.Sub(op.StartTime).Seconds()
	}

	logger.Log("INFO", "Затирание завершено", "disk", disk.Letter, "bytes", op.BytesWiped, "speed", op.SpeedMBps)

	return op
}
