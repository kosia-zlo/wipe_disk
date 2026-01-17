package config

import (
	"fmt"
)

// ApplyProfile применяет профиль производительности к конфигурации
func ApplyProfile(cfg *Config, profile string) error {
	switch profile {
	case "safe":
		cfg.Wipe.MaxSpeedMBps = 10
		cfg.Wipe.ChunkSize = 8 * 1024 * 1024 // 8MB
		cfg.Wipe.FileDelayMs = 500
		cfg.Wipe.SSDPasses = 1
		cfg.Wipe.HDDPasses = 1
	case "balanced":
		cfg.Wipe.MaxSpeedMBps = 25
		cfg.Wipe.ChunkSize = 32 * 1024 * 1024 // 32MB
		cfg.Wipe.FileDelayMs = 200
		cfg.Wipe.SSDPasses = 1
		cfg.Wipe.HDDPasses = 3
	case "aggressive":
		cfg.Wipe.MaxSpeedMBps = 0              // unlimited
		cfg.Wipe.ChunkSize = 128 * 1024 * 1024 // 128MB
		cfg.Wipe.FileDelayMs = 0
		cfg.Wipe.SSDPasses = 2
		cfg.Wipe.HDDPasses = 5
	case "fast":
		cfg.Wipe.MaxSpeedMBps = 0              // unlimited
		cfg.Wipe.ChunkSize = 256 * 1024 * 1024 // 256MB
		cfg.Wipe.FileDelayMs = 0
		cfg.Wipe.SSDPasses = 1
		cfg.Wipe.HDDPasses = 1
	case "sdelete":
		cfg.Wipe.SSDMethod = "sdelete-compatible"
		cfg.Wipe.HDDMethod = "sdelete-compatible"
		cfg.Wipe.SSDPasses = 1
		cfg.Wipe.HDDPasses = 1
		cfg.Wipe.MaxSpeedMBps = 50
		cfg.Wipe.ChunkSize = 64 * 1024 * 1024 // 64MB
		cfg.Wipe.FileDelayMs = 100
		cfg.Wipe.EnableTrim = true
	default:
		return fmt.Errorf("неизвестный профиль: %s", profile)
	}
	return nil
}
