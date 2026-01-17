package wipe

import (
	"context"
	"fmt"
	"time"

	"wipedisk_enterprise/internal/config"
	"wipedisk_enterprise/internal/logging"
	"wipedisk_enterprise/internal/system"
)

// wipeWithStrategy выполняет затирание с использованием стратегии
func wipeWithStrategy(ctx context.Context, disk system.DiskInfo, cfg *config.Config, logger *logging.EnterpriseLogger, dryRun bool, maxDuration time.Duration, mode WipeMode, profile string) *WipeOperation {
	// Создаем конфигурацию для затирания
	wipeConfig := &WipeConfig{
		Passes:       getPassesForMode(cfg, mode),
		MaxSpeedMBps: cfg.Wipe.MaxSpeedMBps,
		MaxDuration:  maxDuration,
	}

	logger.Log("INFO", "Запуск затирания со стратегией", "disk", disk.Letter, "mode", mode, "profile", profile, "passes", wipeConfig.Passes)

	if dryRun {
		op := &WipeOperation{
			ID:        fmt.Sprintf("strategy_%d", time.Now().UnixNano()),
			Disk:      disk.Letter,
			Method:    string(mode),
			Passes:    wipeConfig.Passes,
			ChunkSize: int64(GetStrategy(mode).GetFileSize(disk.Type, profile)),
			Status:    "COMPLETED",
			StartTime: time.Now(),
		}
		now := time.Now()
		op.EndTime = &now
		op.BytesWiped = disk.FreeSize
		op.SpeedMBps = 25.0 // Примерная скорость
		logger.Log("INFO", "DRY RUN: затирание со стратегией завершено", "disk", disk.Letter, "bytes", op.BytesWiped)
		return op
	}

	// Выполняем затирание с стратегией
	op, err := ExecuteWipeWithStrategy(ctx, disk, wipeConfig, logger, mode, profile)
	if err != nil {
		if op == nil {
			op = &WipeOperation{
				ID:        fmt.Sprintf("strategy_%d", time.Now().UnixNano()),
				Disk:      disk.Letter,
				Method:    string(mode),
				Status:    "FAILED",
				StartTime: time.Now(),
				Error:     err.Error(),
			}
		}
		now := time.Now()
		op.EndTime = &now
	}

	return op
}

// getPassesForMode возвращает количество проходов для режима
func getPassesForMode(cfg *config.Config, mode WipeMode) int {
	switch mode {
	case ModeCipher:
		return 3 // Всегда 3 прохода для cipher
	case ModeSDelete:
		return 1 // SDelete использует 1 проход
	case ModeStandard:
		fallthrough
	default:
		// Use SSD passes as base value for standard mode
		return cfg.Wipe.SSDPasses
	}
}

// ValidateMode проверяет корректность режима
func ValidateMode(mode string) (WipeMode, error) {
	m := WipeMode(mode)
	switch m {
	case ModeStandard, ModeSDelete, ModeCipher:
		return m, nil
	default:
		return "", fmt.Errorf("неподдерживаемый режим затирания: %s", mode)
	}
}
