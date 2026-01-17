package security

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"wipedisk_enterprise/internal/config"
	"wipedisk_enterprise/internal/system"
)

func SecurityChecks(cfg *config.Config) error {
	if cfg == nil {
		cfg = config.Default()
	}

	if cfg.Security.RequireAdmin {
		if !IsAdmin() {
			return fmt.Errorf("требуются права администратора")
		}
	}

	if cfg.Security.BlockServers {
		if IsServerOS() {
			return fmt.Errorf("запуск на серверных ОС запрещен")
		}
	}

	return nil
}

// Проверка прав администратора
func IsAdmin() bool {
	if runtime.GOOS != "windows" {
		return false
	}

	return os.Getenv("USERNAME") != "" && os.Getenv("USERDOMAIN") != ""
}

// Проверка на серверную ОС
func IsServerOS() bool {
	serverIndicators := []string{
		"Windows Server",
		"Server",
		"DC",
		"Domain Controller",
	}

	for _, indicator := range serverIndicators {
		if strings.Contains(os.Getenv("OS"), indicator) {
			return true
		}
	}

	return false
}

func ShouldSkipDisk(cfg *config.Config, disk system.DiskInfo) bool {
	if cfg != nil {
		for _, excluded := range cfg.Security.ExcludedDrives {
			if disk.Letter == excluded {
				return true
			}
		}
	}

	// Skip system disk only if not explicitly allowed and not in excluded drives
	if disk.IsSystem {
		if cfg != nil && len(cfg.Security.ExcludedDrives) > 0 {
			// If excluded drives are specified, only skip if C: is in excluded list
			for _, excluded := range cfg.Security.ExcludedDrives {
				if disk.Letter == excluded {
					return true
				}
			}
			return false // Don't skip system disk if excluded drives are configured but C: not in list
		}
		return true // Default behavior: skip system disk
	}

	return false
}
