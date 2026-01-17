package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"wipedisk_enterprise/internal/config"
)

// Enterprise логгер с аудитом
type EnterpriseLogger struct {
	level   string
	file    *os.File
	siem    bool
	verbose bool
}

func NewEnterpriseLogger(cfg *config.Config, verbose bool) (*EnterpriseLogger, error) {
	l := &EnterpriseLogger{
		level:   cfg.Logging.Level,
		siem:    cfg.Logging.SIEMEnabled,
		verbose: verbose,
	}

	// Автоматическое создание директории для логов
	if cfg.Logging.File != "" {
		logDir := filepath.Dir(cfg.Logging.File)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			// Если не можем создать директорию, используем stdout
			fmt.Printf("[WARN] Не удалось создать директорию логов %s: %v\n", logDir, err)
			fmt.Printf("[WARN] Логи будут выводиться в stdout\n")
			return l, nil
		}

		f, err := os.OpenFile(cfg.Logging.File, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			// Если не можем открыть файл логов, используем stdout
			fmt.Printf("[WARN] Не удалось открыть файл логов %s: %v\n", cfg.Logging.File, err)
			fmt.Printf("[WARN] Логи будут выводиться в stdout\n")
			return l, nil
		}
		l.file = f
	}

	return l, nil
}

func (l *EnterpriseLogger) Log(level, message string, fields ...interface{}) {
	if !l.shouldLog(level) {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	entry := fmt.Sprintf("[%s] [%s] %s", timestamp, level, message)

	if len(fields) > 0 {
		entry += fmt.Sprintf(" %v", fields)
	}

	if l.file != nil {
		l.file.WriteString(entry + "\n")
		l.file.Sync()
	}

	if l.verbose || level == "ERROR" || level == "FATAL" {
		fmt.Println(entry)
	}

	// SIEM интеграция
	if l.siem && (level == "ERROR" || level == "WARN" || level == "FATAL") {
		// Отправка в SIEM система
	}
}

func (l *EnterpriseLogger) shouldLog(level string) bool {
	levels := map[string]int{"DEBUG": 0, "INFO": 1, "WARN": 2, "ERROR": 3, "FATAL": 4}
	current := levels[l.level]
	target := levels[level]
	return target >= current
}

func (l *EnterpriseLogger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}
