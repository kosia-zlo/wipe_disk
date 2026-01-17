package reporting

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"wipedisk_enterprise/internal/maintenance"
	"wipedisk_enterprise/internal/system"
)

// VerificationReportJSON расширенная структура для JSON отчёта
type VerificationReportJSON struct {
	maintenance.VerificationReport
	Metadata VerificationMetadata     `json:"metadata"`
	System   system.SystemEnvironment `json:"system"`
}

// VerificationMetadata метаданные верификации
type VerificationMetadata struct {
	RunID       string    `json:"run_id"`
	Timestamp   time.Time `json:"timestamp"`
	Environment string    `json:"environment"`
	Operator    string    `json:"operator"`
	Purpose     string    `json:"purpose"`
}

// GenerateVerificationReport генерирует расширенный отчёт верификации
func GenerateVerificationReport(baseReport *maintenance.VerificationReport, metadata VerificationMetadata) *VerificationReportJSON {
	return &VerificationReportJSON{
		VerificationReport: *baseReport,
		Metadata:           metadata,
		System:             collectSystemInfo(),
	}
}

// SaveVerificationReport сохраняет отчёт верификации
func SaveVerificationReport(report *VerificationReportJSON, format string, outputPath string) error {
	switch format {
	case "json":
		return saveVerificationReportJSON(report, outputPath)
	case "csv":
		return saveVerificationReportCSV(report, outputPath)
	default:
		return fmt.Errorf("неподдерживаемый формат: %s", format)
	}
}

// saveVerificationReportJSON сохраняет отчёт в JSON формате
func saveVerificationReportJSON(report *VerificationReportJSON, outputPath string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации JSON: %w", err)
	}

	if outputPath == "" {
		// Автоматическое имя файла
		timestamp := report.TestDate.Format("20060102_150405")
		outputPath = filepath.Join(os.Getenv("TEMP"), fmt.Sprintf("wipedisk_verification_%s.json", timestamp))
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("ошибка сохранения файла: %w", err)
	}

	return nil
}

// saveVerificationReportCSV сохраняет отчёт в CSV формате
func saveVerificationReportCSV(report *VerificationReportJSON, outputPath string) error {
	if outputPath == "" {
		timestamp := report.TestDate.Format("20060102_150405")
		outputPath = filepath.Join(os.Getenv("TEMP"), fmt.Sprintf("wipedisk_verification_%s.csv", timestamp))
	}

	// Создаем CSV контент
	csvContent := fmt.Sprintf(`# WipeDisk Verification Report
# Generated: %s
# Verification Level: %s
# Disk: %s
# Method: %s
# Passes: %d

Metric,Value
Wipe Verified,%t
Verification Level,%s
Recovery Attempts,%d
Recovered Data (bytes),%d
Test Duration,%s
Success Rate,%.2f%%
Test Date,%s
Anomalies Count,%d
Compliance Standards,"%s"

# Anomalies:
`,
		report.TestDate.Format(time.RFC3339),
		report.VerificationLevel,
		report.Disk,
		report.Method,
		report.Passes,
		report.WipeVerified,
		report.VerificationLevel,
		report.RecoveryAttempts,
		report.RecoveredData,
		report.TestDuration.String(),
		report.SuccessRate,
		report.TestDate.Format(time.RFC3339),
		len(report.Anomalies),
		fmt.Sprintf("%v", report.Compliance),
	)

	// Добавляем аномалии
	for i, anomaly := range report.Anomalies {
		csvContent += fmt.Sprintf("Anomaly %d,%s|%s|%s|%s\n",
			i+1, anomaly.Type, anomaly.Description, anomaly.Location, anomaly.Severity)
	}

	// Добавляем системную информацию
	csvContent += fmt.Sprintf(`
# System Information:
OS Version,%s
Architecture,%s
Username,%s
Domain,%s
Machine Name,%s

# Metadata:
Run ID,%s
Environment,%s
Operator,%s
Purpose,%s
`,
		report.System.OSVersion,
		report.System.Architecture,
		report.System.Username,
		report.System.Domain,
		report.System.MachineName,
		report.Metadata.RunID,
		report.Metadata.Environment,
		report.Metadata.Operator,
		report.Metadata.Purpose,
	)

	if err := os.WriteFile(outputPath, []byte(csvContent), 0644); err != nil {
		return fmt.Errorf("ошибка сохранения CSV файла: %w", err)
	}

	return nil
}

// collectSystemInfo собирает информацию о системе
func collectSystemInfo() system.SystemEnvironment {
	return system.SystemEnvironment{
		OSVersion:    "Windows",
		Architecture: "amd64",
		Username:     os.Getenv("USERNAME"),
		Domain:       os.Getenv("USERDOMAIN"),
		MachineName:  os.Getenv("COMPUTERNAME"),
		IsAdmin:      false,
		IsServer:     false,
		TotalMemory:  0,
		AvailableMem: 0,
		CPUCount:     0,
		Environment:  make(map[string]string),
	}
}

// GenerateRunID генерирует уникальный ID запуска
func GenerateRunID() string {
	return fmt.Sprintf("verify_%d", time.Now().Unix())
}
