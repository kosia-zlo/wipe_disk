package wipe

import (
	"context"
	"fmt"
	"strings"

	"wipedisk_enterprise/internal/logging"
)

type WipeEngine struct {
	wiper  *PersistentFileWiper
	logger *logging.EnterpriseLogger
}

func NewWipeEngine(logger *logging.EnterpriseLogger) *WipeEngine {
	return &WipeEngine{
		wiper:  NewPersistentFileWiper(&PersistentFileConfig{Logger: logger}),
		logger: logger,
	}
}

func (we *WipeEngine) WipeDrive(ctx context.Context, drivePath string, pattern []byte) (*WipeResult, error) {
	// 1. Стерилизация пути
	drivePath = strings.TrimSpace(drivePath)
	drivePath = strings.TrimRight(drivePath, ".")
	drivePath = strings.ToUpper(drivePath)

	if !strings.HasSuffix(drivePath, "\\") {
		if !strings.HasSuffix(drivePath, ":") {
			drivePath += ":"
		}
		drivePath += "\\"
	}

	we.logger.Log("INFO", "Запуск затирания", "drive", drivePath)

	if pattern != nil {
		we.wiper.config.Pattern = pattern
	}

	// 2. Запуск процесса
	result, err := we.wiper.Wipe(ctx, drivePath)
	if err != nil {
		return nil, fmt.Errorf("ошибка при затирании %s: %v", drivePath, err)
	}

	return result, nil
}

func (we *WipeEngine) SetProgressChannel(progress chan<- ProgressInfo) {
	we.wiper.config.Progress = progress
}
