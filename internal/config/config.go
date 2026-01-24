package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Enterprise конфигурация
type Config struct {
	Security struct {
		RequireAdmin        bool     `yaml:"require_admin"`
		BlockServers        bool     `yaml:"block_servers"`
		RequireConfirmation bool     `yaml:"require_confirmation"`
		ExcludedDrives      []string `yaml:"excluded_drives"`
		ProtectedPaths      []string `yaml:"protected_paths"`
	} `yaml:"security"`

	Wipe struct {
		Enabled       bool    `yaml:"enabled"`
		SSDMethod     string  `yaml:"ssd_method"`
		HDDMethod     string  `yaml:"hdd_method"`
		SSDPasses     int     `yaml:"ssd_passes"`
		HDDPasses     int     `yaml:"hdd_passes"`
		ChunkSize     int64   `yaml:"chunk_size"`
		EnableTrim    bool    `yaml:"enable_trim"`
		MaxConcurrent int     `yaml:"max_concurrent"`
		MaxSpeedMBps  float64 `yaml:"max_speed_mbps"`
		MaxDuration   string  `yaml:"max_duration"`
		FileDelayMs   int     `yaml:"file_delay_ms"`
		TargetDrive   string  `yaml:"target_drive"`
	} `yaml:"wipe"`

	Logging struct {
		Level       string `yaml:"level"`
		File        string `yaml:"file"`
		MaxSizeMB   int    `yaml:"max_size_mb"`
		MaxFiles    int    `yaml:"max_files"`
		Structured  bool   `yaml:"structured"`
		SIEMEnabled bool   `yaml:"siem_enabled"`
		SIEMServer  string `yaml:"siem_server"`
		LogPath     string `yaml:"log_path"`
	} `yaml:"logging"`

	Reporting struct {
		Enabled     bool   `yaml:"enabled"`
		LocalPath   string `yaml:"local_path"`
		NetworkPath string `yaml:"network_path"`
		Format      string `yaml:"format"`
	} `yaml:"reporting"`

	Clean struct {
		Enabled         bool     `yaml:"enabled"`
		IncludePaths    []string `yaml:"include_paths"`
		ExcludePaths    []string `yaml:"exclude_paths"`
		ExcludePatterns []string `yaml:"exclude_patterns"`
		MaxFileSize     int64    `yaml:"max_file_size"`
		MinFileAge      int      `yaml:"min_file_age"`
	} `yaml:"clean"`
}

// Default возвращает конфигурацию по умолчанию
func Default() *Config {
	// Get system drive dynamically
	systemDrive := getSystemDrive()

	return &Config{
		Security: struct {
			RequireAdmin        bool     `yaml:"require_admin"`
			BlockServers        bool     `yaml:"block_servers"`
			RequireConfirmation bool     `yaml:"require_confirmation"`
			ExcludedDrives      []string `yaml:"excluded_drives"`
			ProtectedPaths      []string `yaml:"protected_paths"`
		}{
			RequireAdmin:        true,
			BlockServers:        true,
			RequireConfirmation: true,
			ExcludedDrives:      []string{"A:", "B:"},
			ProtectedPaths: []string{
				filepath.Join(systemDrive, "Windows"),
				filepath.Join(systemDrive, "Program Files"),
				filepath.Join(systemDrive, "Program Files (x86)"),
				filepath.Join(systemDrive, "Users"),
			},
		},
		Wipe: struct {
			Enabled       bool    `yaml:"enabled"`
			SSDMethod     string  `yaml:"ssd_method"`
			HDDMethod     string  `yaml:"hdd_method"`
			SSDPasses     int     `yaml:"ssd_passes"`
			HDDPasses     int     `yaml:"hdd_passes"`
			ChunkSize     int64   `yaml:"chunk_size"`
			EnableTrim    bool    `yaml:"enable_trim"`
			MaxConcurrent int     `yaml:"max_concurrent"`
			MaxSpeedMBps  float64 `yaml:"max_speed_mbps"`
			MaxDuration   string  `yaml:"max_duration"`
			FileDelayMs   int     `yaml:"file_delay_ms"`
			TargetDrive   string  `yaml:"target_drive"`
		}{
			Enabled:       true,
			SSDMethod:     "cipher",
			HDDMethod:     "random",
			SSDPasses:     1,
			HDDPasses:     1,
			ChunkSize:     4 * 1024 * 1024, // 4MB
			EnableTrim:    true,
			MaxConcurrent: 2,
			MaxSpeedMBps:  100, // 100MB/s по умолчанию
			MaxDuration:   "2h",
			FileDelayMs:   100,
			TargetDrive:   "",
		},
		Logging: struct {
			Level       string `yaml:"level"`
			File        string `yaml:"file"`
			MaxSizeMB   int    `yaml:"max_size_mb"`
			MaxFiles    int    `yaml:"max_files"`
			Structured  bool   `yaml:"structured"`
			SIEMEnabled bool   `yaml:"siem_enabled"`
			SIEMServer  string `yaml:"siem_server"`
			LogPath     string `yaml:"log_path"`
		}{
			Level:       "INFO",
			File:        "",
			MaxSizeMB:   100,
			MaxFiles:    5,
			Structured:  true,
			SIEMEnabled: false,
			SIEMServer:  "",
			LogPath:     "./logs",
		},
		Reporting: struct {
			Enabled     bool   `yaml:"enabled"`
			LocalPath   string `yaml:"local_path"`
			NetworkPath string `yaml:"network_path"`
			Format      string `yaml:"format"`
		}{
			Enabled:     true,
			LocalPath:   "./reports",
			NetworkPath: "",
			Format:      "json",
		},
		Clean: struct {
			Enabled         bool     `yaml:"enabled"`
			IncludePaths    []string `yaml:"include_paths"`
			ExcludePaths    []string `yaml:"exclude_paths"`
			ExcludePatterns []string `yaml:"exclude_patterns"`
			MaxFileSize     int64    `yaml:"max_file_size"`
			MinFileAge      int      `yaml:"min_file_age"`
		}{
			Enabled:         false,
			IncludePaths:    []string{},
			ExcludePaths:    []string{},
			ExcludePatterns: []string{"*.tmp", "*.temp", "*.log"},
			MaxFileSize:     100 * 1024 * 1024, // 100MB
			MinFileAge:      7,                 // 7 дней
		},
	}
}

// Load загружает конфигурацию из файла
func Load(path string) (*Config, error) {
	if path == "" {
		return Default(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	// Валидация конфигурации
	if err := Validate(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// Validate проверяет конфигурацию на валидность
func Validate(config *Config) error {
	// Валидация security секции
	if config.Security.RequireAdmin && !isAdmin() {
		return fmt.Errorf("configuration requires admin rights")
	}

	// Валидация wipe секции
	if config.Wipe.Enabled {
		// Проверяем passes
		if config.Wipe.SSDPasses <= 0 || config.Wipe.SSDPasses > 10 {
			return fmt.Errorf("SSD passes must be between 1 and 10, got %d", config.Wipe.SSDPasses)
		}
		if config.Wipe.HDDPasses <= 0 || config.Wipe.HDDPasses > 10 {
			return fmt.Errorf("HDD passes must be between 1 and 10, got %d", config.Wipe.HDDPasses)
		}

		// Проверяем chunk size
		if config.Wipe.ChunkSize <= 0 {
			return fmt.Errorf("chunk size must be positive, got %d", config.Wipe.ChunkSize)
		}
		if config.Wipe.ChunkSize > 100*1024*1024 { // 100MB max
			return fmt.Errorf("chunk size too large (max 100MB), got %d", config.Wipe.ChunkSize)
		}

		// Проверяем concurrent operations
		if config.Wipe.MaxConcurrent <= 0 || config.Wipe.MaxConcurrent > 10 {
			return fmt.Errorf("max concurrent must be between 1 and 10, got %d", config.Wipe.MaxConcurrent)
		}

		// Проверяем speed
		if config.Wipe.MaxSpeedMBps < 0 {
			return fmt.Errorf("max speed cannot be negative, got %f", config.Wipe.MaxSpeedMBps)
		}
		if config.Wipe.MaxSpeedMBps > 1000 { // 1GB/s max
			return fmt.Errorf("max speed too high (max 1000MB/s), got %f", config.Wipe.MaxSpeedMBps)
		}

		// Проверяем duration
		if config.Wipe.MaxDuration != "" {
			if _, err := time.ParseDuration(config.Wipe.MaxDuration); err != nil {
				return fmt.Errorf("invalid max duration format: %s", config.Wipe.MaxDuration)
			}
		}

		// Проверяем file delay
		if config.Wipe.FileDelayMs < 0 || config.Wipe.FileDelayMs > 60000 { // max 60 seconds
			return fmt.Errorf("file delay must be between 0 and 60000ms, got %d", config.Wipe.FileDelayMs)
		}

		// Валидация методов
		validMethods := map[string]bool{
			"random": true,
			"zeros":  true,
			"cipher": true,
		}
		if !validMethods[config.Wipe.SSDMethod] {
			return fmt.Errorf("invalid SSD method: %s", config.Wipe.SSDMethod)
		}
		if !validMethods[config.Wipe.HDDMethod] {
			return fmt.Errorf("invalid HDD method: %s", config.Wipe.HDDMethod)
		}
	}

	// Валидация logging секции
	validLevels := map[string]bool{
		"DEBUG": true,
		"INFO":  true,
		"WARN":  true,
		"ERROR": true,
	}
	if !validLevels[config.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", config.Logging.Level)
	}

	if config.Logging.MaxSizeMB <= 0 || config.Logging.MaxSizeMB > 1000 {
		return fmt.Errorf("log max size must be between 1MB and 1000MB, got %d", config.Logging.MaxSizeMB)
	}

	if config.Logging.MaxFiles <= 0 || config.Logging.MaxFiles > 50 {
		return fmt.Errorf("log max files must be between 1 and 50, got %d", config.Logging.MaxFiles)
	}

	// Валидация путей
	for _, path := range config.Security.ProtectedPaths {
		if path == "" {
			return fmt.Errorf("empty protected path")
		}

		absPath := filepath.Clean(path)
		if absPath == "" || absPath == "." || absPath == "/" {
			return fmt.Errorf("invalid protected path: %s", path)
		}
	}

	return nil
}

// Save сохраняет конфигурацию в файл
func Save(config *Config, path string) error {
	// Валидация перед сохранением
	if err := Validate(config); err != nil {
		return fmt.Errorf("cannot save invalid config: %w", err)
	}

	// Создаем директорию если нужно
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetMaxDuration возвращает максимальную длительность
func (config *Config) GetMaxDuration() time.Duration {
	if config.Wipe.MaxDuration == "" {
		return 0 // Без лимита
	}

	duration, err := time.ParseDuration(config.Wipe.MaxDuration)
	if err != nil {
		return 2 * time.Hour // Fallback
	}

	return duration
}

// isAdmin проверяет права администратора
func isAdmin() bool {
	// Упрощенная проверка
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	return err == nil
}

// getSystemDrive возвращает системный диск (C:, D:, и т.д.)
func getSystemDrive() string {
	// Получаем путь к системной директории
	windir := os.Getenv("WINDIR")
	if windir == "" {
		return "C:" // Fallback
	}

	// Извлекаем букву диска
	if len(windir) >= 2 {
		return windir[:2]
	}

	return "C:" // Fallback
}
