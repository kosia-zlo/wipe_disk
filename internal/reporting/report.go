package reporting

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"wipedisk_enterprise/internal/config"
	"wipedisk_enterprise/internal/wipe"
)

// Report представляет JSON отчёт о запуске
type Report struct {
	RunID       string                 `json:"run_id"`
	Version     string                 `json:"version"`
	Timestamp   time.Time              `json:"timestamp"`
	Config      map[string]interface{} `json:"config"`
	Engine      string                 `json:"engine"`
	Profile     string                 `json:"profile,omitempty"`
	DryRun      bool                   `json:"dry_run"`
	MaxDuration string                 `json:"max_duration,omitempty"`
	Operations  []OperationReport      `json:"operations"`
	Summary     SummaryReport          `json:"summary"`
	ExitCode    int                    `json:"exit_code"`
	Duration    string                 `json:"duration"`
}

// OperationReport представляет отчёт об операции затирания
type OperationReport struct {
	ID         string     `json:"id"`
	Disk       string     `json:"disk"`
	Method     string     `json:"method"`
	Passes     int        `json:"passes"`
	ChunkSize  int64      `json:"chunk_size"`
	Status     string     `json:"status"`
	StartTime  time.Time  `json:"start_time"`
	EndTime    *time.Time `json:"end_time,omitempty"`
	BytesWiped uint64     `json:"bytes_wiped"`
	SpeedMBps  float64    `json:"speed_mbps"`
	Error      string     `json:"error,omitempty"`
	Warning    string     `json:"warning,omitempty"`
}

// SummaryReport представляет сводную информацию
type SummaryReport struct {
	TotalDisks   int     `json:"total_disks"`
	Completed    int     `json:"completed"`
	Partial      int     `json:"partial"`
	Cancelled    int     `json:"cancelled"`
	Failed       int     `json:"failed"`
	TotalBytes   uint64  `json:"total_bytes"`
	AverageSpeed float64 `json:"average_speed_mbps"`
	SuccessRate  float64 `json:"success_rate"`
}

// AggregatedReport представляет агрегированный отчёт по нескольким запускам
type AggregatedReport struct {
	GeneratedAt   time.Time         `json:"generated_at"`
	Reports       []Report          `json:"reports"`
	TotalRuns     int               `json:"total_runs"`
	TotalMachines int               `json:"total_machines"` // TODO: добавить hostname
	TotalDisks    int               `json:"total_disks"`
	TotalBytes    uint64            `json:"total_bytes"`
	Summary       AggregatedSummary `json:"summary"`
	SuccessRate   float64           `json:"overall_success_rate"`
}

// AggregatedSummary представляет агрегированную сводку
type AggregatedSummary struct {
	Completed  int     `json:"completed"`
	Partial    int     `json:"partial"`
	Cancelled  int     `json:"cancelled"`
	Failed     int     `json:"failed"`
	SuccessPct float64 `json:"success_pct"`
}

// GenerateReport генерирует JSON отчёт о запуске
func GenerateReport(operations []*wipe.WipeOperation, cfg *config.Config, engine, profile string, dryRun bool, maxDuration time.Duration, startTime, endTime time.Time, exitCode int) (*Report, error) {
	report := &Report{
		RunID:       fmt.Sprintf("run_%d", startTime.UnixNano()),
		Version:     "1.2.2",
		Timestamp:   startTime,
		Config:      configToMap(cfg),
		Engine:      engine,
		Profile:     profile,
		DryRun:      dryRun,
		MaxDuration: maxDuration.String(),
		Operations:  make([]OperationReport, len(operations)),
		Summary:     SummaryReport{},
		ExitCode:    exitCode,
		Duration:    endTime.Sub(startTime).String(),
	}

	var totalBytes uint64
	var totalSpeed float64
	completed := 0
	partial := 0
	cancelled := 0
	failed := 0

	for i, op := range operations {
		opReport := OperationReport{
			ID:         op.ID,
			Disk:       op.Disk,
			Method:     op.Method,
			Passes:     op.Passes,
			ChunkSize:  op.ChunkSize,
			Status:     op.Status,
			StartTime:  op.StartTime,
			BytesWiped: op.BytesWiped,
			SpeedMBps:  op.SpeedMBps,
		}

		if op.EndTime != nil {
			opReport.EndTime = op.EndTime
		}

		if op.Error != "" {
			opReport.Error = op.Error
			failed++
		} else if op.Warning != "" {
			opReport.Warning = op.Warning
			partial++
		} else if op.Status == "completed" {
			completed++
		} else if op.Status == "cancelled" {
			cancelled++
		}

		totalBytes += op.BytesWiped
		totalSpeed += op.SpeedMBps

		report.Operations[i] = opReport
	}

	report.Summary = SummaryReport{
		TotalDisks:   len(operations),
		Completed:    completed,
		Partial:      partial,
		Cancelled:    cancelled,
		Failed:       failed,
		TotalBytes:   totalBytes,
		AverageSpeed: totalSpeed / float64(len(operations)),
		SuccessRate:  float64(completed) / float64(len(operations)) * 100,
	}

	return report, nil
}

// GenerateEnterpriseReportWrapper generates both legacy and enterprise reports
func GenerateEnterpriseReportWrapper(operations []*wipe.WipeOperation, cfg *config.Config, engine, profile string, dryRun bool, maxDuration time.Duration, startTime, endTime time.Time, exitCode int) (*EnterpriseReport, error) {
	return GenerateEnterpriseReport(operations, cfg, engine, profile, dryRun, maxDuration, startTime, endTime, exitCode)
}

// SaveEnterpriseReport saves the enterprise report
func SaveEnterpriseReportWrapper(report *EnterpriseReport, cfg *config.Config, format string) error {
	return SaveEnterpriseReport(report, cfg, format)
}

// SaveReport сохраняет отчёт в JSON файл
func SaveReport(report *Report, cfg *config.Config) error {
	if !cfg.Reporting.Enabled {
		return nil
	}

	// Создаем директорию для отчётов
	if err := os.MkdirAll(cfg.Reporting.LocalPath, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории для отчётов: %w", err)
	}

	// Имя файла отчёта
	filename := fmt.Sprintf("wipedisk_report_%s.json", report.Timestamp.Format("20060102_150405"))
	filepath := filepath.Join(cfg.Reporting.LocalPath, filename)

	// Сериализация в JSON
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации отчёта: %w", err)
	}

	// Запись в файл
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("ошибка записи отчёта: %w", err)
	}

	return nil
}

// AggregateReports агрегирует несколько отчётов в один
func AggregateReports(reports []Report) *AggregatedReport {
	agg := &AggregatedReport{
		GeneratedAt: time.Now(),
		Reports:     reports,
		TotalRuns:   len(reports),
		TotalDisks:  0,
		TotalBytes:  0,
		Summary:     AggregatedSummary{},
	}

	machines := make(map[string]bool)
	completed := 0
	partial := 0
	cancelled := 0
	failed := 0

	for _, report := range reports {
		agg.TotalDisks += report.Summary.TotalDisks
		agg.TotalBytes += report.Summary.TotalBytes

		// TODO: добавить hostname в отчёт
		machines["unknown"] = true

		completed += report.Summary.Completed
		partial += report.Summary.Partial
		cancelled += report.Summary.Cancelled
		failed += report.Summary.Failed
	}

	agg.TotalMachines = len(machines)
	agg.Summary = AggregatedSummary{
		Completed:  completed,
		Partial:    partial,
		Cancelled:  cancelled,
		Failed:     failed,
		SuccessPct: float64(completed) / float64(completed+partial+cancelled+failed) * 100,
	}

	agg.SuccessRate = agg.Summary.SuccessPct

	return agg
}

// SaveAggregatedReport сохраняет агрегированный отчёт
func SaveAggregatedReport(agg *AggregatedReport, cfg *config.Config) error {
	if !cfg.Reporting.Enabled {
		return nil
	}

	// Создаем директорию для отчётов
	if err := os.MkdirAll(cfg.Reporting.LocalPath, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории для отчётов: %w", err)
	}

	// Имя файла агрегированного отчёта
	filename := fmt.Sprintf("wipedisk_aggregated_%s.json", agg.GeneratedAt.Format("20060102_150405"))
	filepath := filepath.Join(cfg.Reporting.LocalPath, filename)

	// Сериализация в JSON
	data, err := json.MarshalIndent(agg, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации агрегированного отчёта: %w", err)
	}

	// Запись в файл
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("ошибка записи агрегированного отчёта: %w", err)
	}

	return nil
}

// configToMap преобразует Config в map для JSON сериализации
func configToMap(cfg *config.Config) map[string]interface{} {
	return map[string]interface{}{
		"security": map[string]interface{}{
			"require_admin":        cfg.Security.RequireAdmin,
			"block_servers":        cfg.Security.BlockServers,
			"require_confirmation": cfg.Security.RequireConfirmation,
			"excluded_drives":      cfg.Security.ExcludedDrives,
			"protected_paths":      cfg.Security.ProtectedPaths,
		},
		"wipe": map[string]interface{}{
			"enabled":        cfg.Wipe.Enabled,
			"ssd_method":     cfg.Wipe.SSDMethod,
			"hdd_method":     cfg.Wipe.HDDMethod,
			"ssd_passes":     cfg.Wipe.SSDPasses,
			"hdd_passes":     cfg.Wipe.HDDPasses,
			"chunk_size":     cfg.Wipe.ChunkSize,
			"enable_trim":    cfg.Wipe.EnableTrim,
			"max_concurrent": cfg.Wipe.MaxConcurrent,
			"max_speed_mbps": cfg.Wipe.MaxSpeedMBps,
			"file_delay_ms":  cfg.Wipe.FileDelayMs,
			"max_duration":   cfg.Wipe.MaxDuration,
		},
		"clean": map[string]interface{}{
			"enabled":          cfg.Clean.Enabled,
			"include_paths":    cfg.Clean.IncludePaths,
			"exclude_paths":    cfg.Clean.ExcludePaths,
			"exclude_patterns": cfg.Clean.ExcludePatterns,
			"max_file_size":    cfg.Clean.MaxFileSize,
			"min_file_age":     cfg.Clean.MinFileAge,
		},
		"logging": map[string]interface{}{
			"level":        cfg.Logging.Level,
			"file":         cfg.Logging.File,
			"max_size":     cfg.Logging.MaxSizeMB,
			"max_backups":  cfg.Logging.MaxFiles,
			"siem_enabled": cfg.Logging.SIEMEnabled,
			"siem_server":  cfg.Logging.SIEMServer,
		},
		"reporting": map[string]interface{}{
			"enabled":      cfg.Reporting.Enabled,
			"local_path":   cfg.Reporting.LocalPath,
			"network_path": cfg.Reporting.NetworkPath,
			"format":       cfg.Reporting.Format,
		},
	}
}
