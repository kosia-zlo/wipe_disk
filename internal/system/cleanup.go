package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CleanupOperation represents a cleanup operation
type CleanupOperation struct {
	Name        string
	Description string
	Category    string
	RiskLevel   string
	Execute     func() error
}

// GetCleanupOperations returns all available cleanup operations
func GetCleanupOperations() []CleanupOperation {
	return []CleanupOperation{
		{
			Name:        "Очистка очереди печати",
			Description: "Остановка службы печати, очистка очереди, перезапуск службы",
			Category:    "Системные артефакты",
			RiskLevel:   "Низкий",
			Execute:     ClearPrintQueue,
		},
		{
			Name:        "Очистка DNS кэша",
			Description: "Сброс DNS кэша, регистрация DNS, сброс winsock",
			Category:    "Системные артефакты",
			RiskLevel:   "Низкий",
			Execute:     ClearDNSCache,
		},
		{
			Name:        "Очистка кэша браузеров",
			Description: "Удаление кэша и cookies из Chrome, Firefox, Yandex Browser",
			Category:    "Временные файлы",
			RiskLevel:   "Средний",
			Execute:     CleanupBrowserCache,
		},
		{
			Name:        "Очистка старых логов",
			Description: "Удаление логов старше 7 дней",
			Category:    "Файлы журналов",
			RiskLevel:   "Низкий",
			Execute:     func() error { return CleanupOldLogs(7) },
		},
		{
			Name:        "Очистка временных файлов",
			Description: "Удаление временных файлов системы и пользователя",
			Category:    "Временные файлы",
			RiskLevel:   "Средний",
			Execute:     CleanupTempFiles,
		},
	}
}

// ClearPrintQueue clears the print queue
func ClearPrintQueue() error {
	// Остановка службы печати
	if err := exec.Command("net", "stop", "spooler").Run(); err != nil {
		return fmt.Errorf("ошибка остановки службы печати: %w", err)
	}

	// Очистка директории печати
	printDir := filepath.Join(os.Getenv("systemroot"), "System32", "spool", "PRINTERS")
	if err := os.RemoveAll(printDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("ошибка очистки очереди печати: %w", err)
	}

	// Создание пустой директории
	if err := os.MkdirAll(printDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории печати: %w", err)
	}

	// Запуск службы печати
	if err := exec.Command("net", "start", "spooler").Run(); err != nil {
		return fmt.Errorf("ошибка запуска службы печати: %w", err)
	}

	return nil
}

// ClearDNSCache clears DNS cache
func ClearDNSCache() error {
	commands := [][]string{
		{"ipconfig", "/flushdns"},
		{"ipconfig", "/registerdns"},
		{"netsh", "winsock", "reset"},
	}

	for _, cmd := range commands {
		if err := exec.Command(cmd[0], cmd[1:]...).Run(); err != nil {
			return fmt.Errorf("ошибка выполнения команды %s: %w", strings.Join(cmd, " "), err)
		}
	}
	return nil
}

// BrowserCleanup represents browser cleanup targets
type BrowserCleanup struct {
	Name        string
	CachePaths  []string
	CookiesPath string
}

// GetBrowserCleanupTargets returns browser cleanup targets
func GetBrowserCleanupTargets() []BrowserCleanup {
	return []BrowserCleanup{
		{
			Name: "Google Chrome",
			CachePaths: []string{
				filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "User Data", "Default", "Cache"),
				filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "User Data", "Default", "Code Cache"),
				filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "User Data", "Default", "Service Worker"),
			},
			CookiesPath: filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "User Data", "Default", "Cookies"),
		},
		{
			Name: "Mozilla Firefox",
			CachePaths: []string{
				filepath.Join(os.Getenv("APPDATA"), "Mozilla", "Firefox", "Profiles"),
			},
		},
		{
			Name: "Yandex Browser",
			CachePaths: []string{
				filepath.Join(os.Getenv("LOCALAPPDATA"), "Yandex", "YandexBrowser", "User Data", "Default", "Cache"),
				filepath.Join(os.Getenv("LOCALAPPDATA"), "Yandex", "YandexBrowser", "User Data", "Default", "Code Cache"),
				filepath.Join(os.Getenv("LOCALAPPDATA"), "Yandex", "YandexBrowser", "User Data", "Default", "Service Worker"),
			},
			CookiesPath: filepath.Join(os.Getenv("LOCALAPPDATA"), "Yandex", "YandexBrowser", "User Data", "Default", "Cookies"),
		},
	}
}

// CleanupBrowserCache cleans browser cache and cookies
func CleanupBrowserCache() error {
	browsers := GetBrowserCleanupTargets()

	for _, browser := range browsers {
		// Очистка кэша
		for _, path := range browser.CachePaths {
			if err := os.RemoveAll(path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("ошибка очистки кэша %s (%s): %w", browser.Name, path, err)
			}
		}

		// Очистка cookies
		if browser.CookiesPath != "" {
			if err := os.Remove(browser.CookiesPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("ошибка удаления cookies %s: %w", browser.Name, err)
			}
		}
	}
	return nil
}

// CleanupOldLogs removes old log files
func CleanupOldLogs(days int) error {
	// Get system drive dynamically
	systemDrive := getSystemDrive()
	logsDir := filepath.Join(systemDrive, "Windows", "Logs")
	cutoff := time.Now().AddDate(0, 0, -days)

	var errors []string

	err := filepath.Walk(logsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Продолжаем при ошибках доступа
			return nil
		}

		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), ".log") {
			if info.ModTime().Before(cutoff) {
				if err := os.Remove(path); err != nil {
					errors = append(errors, fmt.Sprintf("Ошибка удаления %s: %v", path, err))
				}
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("ошибка обхода директории логов: %w", err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("ошибки при удалении логов: %s", strings.Join(errors, "; "))
	}

	return nil
}

// CleanupTempFiles removes temporary files
func CleanupTempFiles() error {
	tempDirs := []string{
		os.Getenv("TEMP"),
		filepath.Join(os.Getenv("systemroot"), "Temp"),
	}

	for _, tempDir := range tempDirs {
		if tempDir == "" {
			continue
		}

		// Удаление файлов
		err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if path != tempDir && !info.IsDir() {
				if err := os.Remove(path); err != nil {
					return fmt.Errorf("ошибка удаления временного файла %s: %w", path, err)
				}
			}
			return nil
		})

		if err != nil {
			return fmt.Errorf("ошибка очистки временной директории %s: %w", tempDir, err)
		}
	}

	return nil
}

// ExecuteCleanupOperations executes specified cleanup operations
func ExecuteCleanupOperations(operations []string) ([]CleanupResult, error) {
	var results []CleanupResult
	availableOps := GetCleanupOperations()

	for _, opName := range operations {
		var targetOp *CleanupOperation
		for i := range availableOps {
			if availableOps[i].Name == opName {
				targetOp = &availableOps[i]
				break
			}
		}

		if targetOp == nil {
			return results, fmt.Errorf("операция очистки не найдена: %s", opName)
		}

		result := CleanupResult{
			Name:        targetOp.Name,
			Description: targetOp.Description,
			Category:    targetOp.Category,
			RiskLevel:   targetOp.RiskLevel,
			StartTime:   time.Now(),
		}

		if err := targetOp.Execute(); err != nil {
			result.Error = err.Error()
			result.Status = "FAILED"
		} else {
			result.Status = "COMPLETED"
		}

		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)

		results = append(results, result)
	}

	return results, nil
}

// CleanupResult represents the result of a cleanup operation
type CleanupResult struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Category    string        `json:"category"`
	RiskLevel   string        `json:"risk_level"`
	Status      string        `json:"status"`
	Error       string        `json:"error,omitempty"`
	StartTime   time.Time     `json:"start_time"`
	EndTime     time.Time     `json:"end_time"`
	Duration    time.Duration `json:"duration"`
}
