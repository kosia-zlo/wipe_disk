package wipe

import (
	"context"
	"fmt"

	"wipedisk_enterprise/internal/logging"
	"wipedisk_enterprise/internal/system"
)

// WipeEngine provides unified interface for wipe operations
type WipeEngine struct {
	wiper  *PersistentFileWiper
	logger *logging.EnterpriseLogger
}

// NewWipeEngine creates new wipe engine
func NewWipeEngine(logger *logging.EnterpriseLogger) *WipeEngine {
	return &WipeEngine{
		wiper:  NewPersistentFileWiper(&PersistentFileConfig{Logger: logger}),
		logger: logger,
	}
}

// WipeDrive wipes free space on specified drive
func (we *WipeEngine) WipeDrive(ctx context.Context, drivePath string, pattern []byte) (*WipeResult, error) {
	// Validate drive path
	driveInfo, err := system.GetDiskInfoForPath(drivePath)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения информации о диске: %w", err)
	}

	if !driveInfo.IsWritable {
		return nil, fmt.Errorf("диск %s недоступен для записи", drivePath)
	}

	we.logger.Log("INFO", "Начало затирания диска", "drive", drivePath, "free_space", driveInfo.FreeSize)

	// Update wiper pattern if provided
	if pattern != nil {
		we.wiper.config.Pattern = pattern
	}

	return we.wiper.Wipe(ctx, drivePath)
}

// SetProgressChannel sets progress channel for wipe operations
func (we *WipeEngine) SetProgressChannel(progress chan<- ProgressInfo) {
	we.wiper.config.Progress = progress
}
