package reporting

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"wipedisk_enterprise/internal/config"
	"wipedisk_enterprise/internal/wipe"
)

// RiskLevel represents the risk level
type RiskLevel string

const (
	RiskLow      RiskLevel = "Низкий"
	RiskMedium   RiskLevel = "Средний"
	RiskHigh     RiskLevel = "Высокий"
	RiskCritical RiskLevel = "Критический"
)

// EnterpriseReport represents a professional enterprise security audit report
type EnterpriseReport struct {
	Metadata        ReportMetadata    `json:"metadata"`
	Executive       ExecutiveSummary  `json:"executive_summary"`
	SummaryTable    SummaryTable      `json:"summary_table"`
	Sections        []ReportSection   `json:"sections"`
	Critical        []CriticalFinding `json:"critical_findings"`
	Recommendations []Recommendation  `json:"recommendations"`
	GeneratedAt     time.Time         `json:"generated_at"`
}

// ReportMetadata contains professional header information
type ReportMetadata struct {
	ProductName string    `json:"product_name"`
	ReportTitle string    `json:"report_title"`
	Hostname    string    `json:"hostname"`
	ScanDate    time.Time `json:"scan_date"`
	ScanMode    string    `json:"scan_mode"`
	Version     string    `json:"version"`
	OverallRisk RiskLevel `json:"overall_risk"`
	RunID       string    `json:"run_id"`
	Duration    string    `json:"duration"`
}

// ExecutiveSummary contains high-level analysis
type ExecutiveSummary struct {
	TotalFindings    int               `json:"total_findings"`
	SecurityPosture  string            `json:"security_posture"`
	ComplianceRisk   string            `json:"compliance_risk"`
	RiskDistribution map[RiskLevel]int `json:"risk_distribution"`
	AnalyzedDisks    int               `json:"analyzed_disks"`
	TotalDataWiped   string            `json:"total_data_wiped"`
	SuccessRate      float64           `json:"success_rate"`
}

// SummaryTable represents the summary table by category
type SummaryTable struct {
	Categories []CategorySummary `json:"categories"`
}

// CategorySummary represents summary for each category
type CategorySummary struct {
	CategoryName  string    `json:"category_name"`
	FindingsCount int       `json:"findings_count"`
	RiskLevel     RiskLevel `json:"risk_level"`
	Description   string    `json:"description"`
}

// ReportSection represents a detailed section for each category
type ReportSection struct {
	CategoryName    string        `json:"category_name"`
	Description     string        `json:"description"`
	RiskExplanation string        `json:"risk_explanation"`
	TotalFiles      int           `json:"total_files"`
	RiskLevel       RiskLevel     `json:"risk_level"`
	TopExamples     []FileExample `json:"top_examples"`
	Recommendations []string      `json:"recommendations"`
}

// FileExample represents a critical file example
type FileExample struct {
	Path         string    `json:"path"`
	Size         string    `json:"size"`
	ModifiedDate time.Time `json:"modified_date"`
	RiskFactors  []string  `json:"risk_factors"`
}

// CriticalFinding represents findings requiring immediate attention
type CriticalFinding struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Impact      string    `json:"impact"`
	Location    string    `json:"location"`
	Discovered  time.Time `json:"discovered"`
	Urgency     RiskLevel `json:"urgency"`
}

// Recommendation represents a security recommendation
type Recommendation struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Priority    RiskLevel `json:"priority"`
	Category    string    `json:"category"`
	Effort      string    `json:"effort"`
	Deadline    string    `json:"deadline"`
}

// SecurityCategory represents different security analysis categories
type SecurityCategory struct {
	Name        string
	Description string
	RiskFactors []string
	RiskLevel   RiskLevel
	Files       []FileExample
}

// GenerateEnterpriseReport creates a professional enterprise security audit report
func GenerateEnterpriseReport(operations []*wipe.WipeOperation, cfg *config.Config, engine, profile string, dryRun bool, maxDuration time.Duration, startTime, endTime time.Time, exitCode int) (*EnterpriseReport, error) {
	hostname, _ := os.Hostname()

	report := &EnterpriseReport{
		Metadata: ReportMetadata{
			ProductName: "WipeDisk Enterprise",
			ReportTitle: "Отчет аудита безопасности",
			Hostname:    hostname,
			ScanDate:    startTime,
			ScanMode:    getScanMode(dryRun, profile),
			Version:     "1.2.2",
			RunID:       fmt.Sprintf("run_%d", startTime.UnixNano()),
			Duration:    endTime.Sub(startTime).String(),
		},
		GeneratedAt: time.Now(),
	}

	// Analyze operations and generate findings
	categories := analyzeOperations(operations)

	// Calculate overall risk level
	report.Metadata.OverallRisk = calculateOverallRisk(categories)

	// Generate executive summary
	report.Executive = generateExecutiveSummary(operations, categories)

	// Generate summary table
	report.SummaryTable = generateSummaryTable(categories)

	// Generate detailed sections
	report.Sections = generateReportSections(categories)

	// Identify critical findings
	report.Critical = identifyCriticalFindings(categories)

	// Generate recommendations
	report.Recommendations = generateRecommendations(categories)

	return report, nil
}

// analyzeOperations analyzes wipe operations and categorizes findings
func analyzeOperations(operations []*wipe.WipeOperation) []SecurityCategory {
	var categories []SecurityCategory

	// Category 1: Data Remnants
	dataRemnants := SecurityCategory{
		Name:        "Остатки данных",
		Description: "Файлы или остатки данных, которые были полностью стерты",
		RiskFactors: []string{"Неполное уничтожение данных", "Потенциальное восстановление данных", "Несоответствие политикам хранения данных"},
		RiskLevel:   RiskMedium,
		Files:       []FileExample{},
	}

	// Category 2: System Artifacts
	systemArtifacts := SecurityCategory{
		Name:        "Системные артефакты",
		Description: "Системные файлы и артефакты, которые могут содержать конфиденциальную информацию",
		RiskFactors: []string{"Раскрытие конфигурации системы", "Следы активности пользователей", "Потенциальные форензик данные"},
		RiskLevel:   RiskHigh,
		Files:       []FileExample{},
	}

	// Category 3: Temporary Files
	tempFiles := SecurityCategory{
		Name:        "Временные файлы",
		Description: "Временные файлы, которые могут содержать фрагменты конфиденциальных данных",
		RiskFactors: []string{"Утечка данных через временные файлы", "Раскрытие данных приложений", "Остатки дампов памяти"},
		RiskLevel:   RiskMedium,
		Files:       []FileExample{},
	}

	// Category 4: Log Files
	logFiles := SecurityCategory{
		Name:        "Файлы журналов",
		Description: "Файлы журналов, которые могут содержать конфиденциальные операционные данные",
		RiskFactors: []string{"Раскрытие паттернов активности", "Отслеживание поведения пользователей", "Раскрытие информации о системе"},
		RiskLevel:   RiskLow,
		Files:       []FileExample{},
	}

	// Category 5: Cache Files
	cacheFiles := SecurityCategory{
		Name:        "Файлы кэша",
		Description: "Файлы кэша, которые могут содержать частичную конфиденциальную информацию",
		RiskFactors: []string{"Частичное восстановление данных", "Раскрытие состояния приложений", "Утечка пользовательских настроек"},
		RiskLevel:   RiskLow,
		Files:       []FileExample{},
	}

	// Category 6: System Cleanup
	systemCleanup := SecurityCategory{
		Name:        "Системная очистка",
		Description: "Операции системной очистки для удаления временных данных и артефактов",
		RiskFactors: []string{"Остатки системных операций", "Накопленные временные файлы", "Артефакты сетевой активности"},
		RiskLevel:   RiskMedium,
		Files:       []FileExample{},
	}

	// Analyze each operation and categorize findings
	for _, op := range operations {
		if op.Error != "" {
			// Failed operations are critical findings
			dataRemnants.Files = append(dataRemnants.Files, FileExample{
				Path:         op.Disk,
				Size:         formatBytes(op.BytesWiped),
				ModifiedDate: op.StartTime,
				RiskFactors:  []string{"Сбой операции стирания", "Данные потенциально не стерты"},
			})
			dataRemnants.RiskLevel = RiskCritical
		} else if op.Warning != "" {
			// Operations with warnings
			systemArtifacts.Files = append(systemArtifacts.Files, FileExample{
				Path:         op.Disk,
				Size:         formatBytes(op.BytesWiped),
				ModifiedDate: op.StartTime,
				RiskFactors:  []string{"Частичное стирание", "Предупреждение во время операции"},
			})
		} else {
			// Successful operations - check for potential issues
			if op.BytesWiped < 1024*1024 { // Less than 1MB
				tempFiles.Files = append(tempFiles.Files, FileExample{
					Path:         op.Disk,
					Size:         formatBytes(op.BytesWiped),
					ModifiedDate: op.StartTime,
					RiskFactors:  []string{"Малый размер данных", "Потенциально неполное стирание"},
				})
			}
		}
	}

	categories = []SecurityCategory{dataRemnants, systemArtifacts, tempFiles, logFiles, cacheFiles, systemCleanup}

	// Sort files within each category by risk (largest first)
	for i := range categories {
		sort.Slice(categories[i].Files, func(j, k int) bool {
			return getSizeInBytes(categories[i].Files[j].Size) > getSizeInBytes(categories[i].Files[k].Size)
		})

		// Keep only top 5 examples
		if len(categories[i].Files) > 5 {
			categories[i].Files = categories[i].Files[:5]
		}
	}

	return categories
}

// calculateOverallRisk determines the overall risk level
func calculateOverallRisk(categories []SecurityCategory) RiskLevel {
	hasCritical := false
	hasHigh := false

	for _, cat := range categories {
		if cat.RiskLevel == RiskCritical {
			hasCritical = true
		} else if cat.RiskLevel == RiskHigh {
			hasHigh = true
		}
	}

	if hasCritical {
		return RiskCritical
	} else if hasHigh {
		return RiskHigh
	} else if len(categories) > 0 {
		return RiskMedium
	}

	return RiskLow
}

// generateExecutiveSummary creates the executive summary
func generateExecutiveSummary(operations []*wipe.WipeOperation, categories []SecurityCategory) ExecutiveSummary {
	totalFindings := 0
	riskDist := make(map[RiskLevel]int)

	for _, cat := range categories {
		totalFindings += len(cat.Files)
		riskDist[cat.RiskLevel] += len(cat.Files)
	}

	var totalBytes uint64
	completed := 0
	for _, op := range operations {
		totalBytes += op.BytesWiped
		if op.Status == "completed" {
			completed++
		}
	}

	successRate := float64(completed) / float64(len(operations)) * 100

	securityPosture := "Защищено"
	complianceRisk := "Низкий"

	if riskDist[RiskCritical] > 0 {
		securityPosture = "Критический"
		complianceRisk = "Высокий"
	} else if riskDist[RiskHigh] > 0 {
		securityPosture = "Под риском"
		complianceRisk = "Средний"
	} else if riskDist[RiskMedium] > 0 {
		securityPosture = "Умеренный"
		complianceRisk = "Низкий"
	}

	return ExecutiveSummary{
		TotalFindings:    totalFindings,
		SecurityPosture:  securityPosture,
		ComplianceRisk:   complianceRisk,
		RiskDistribution: riskDist,
		AnalyzedDisks:    len(operations),
		TotalDataWiped:   formatBytes(totalBytes),
		SuccessRate:      successRate,
	}
}

// generateSummaryTable creates the summary table
func generateSummaryTable(categories []SecurityCategory) SummaryTable {
	var table []CategorySummary

	for _, cat := range categories {
		table = append(table, CategorySummary{
			CategoryName:  cat.Name,
			FindingsCount: len(cat.Files),
			RiskLevel:     cat.RiskLevel,
			Description:   cat.Description,
		})
	}

	return SummaryTable{Categories: table}
}

// generateReportSections creates detailed sections
func generateReportSections(categories []SecurityCategory) []ReportSection {
	var sections []ReportSection

	for _, cat := range categories {
		section := ReportSection{
			CategoryName:    cat.Name,
			Description:     cat.Description,
			RiskExplanation: strings.Join(cat.RiskFactors, "; "),
			TotalFiles:      len(cat.Files),
			RiskLevel:       cat.RiskLevel,
			TopExamples:     cat.Files,
			Recommendations: generateCategoryRecommendations(cat),
		}
		sections = append(sections, section)
	}

	return sections
}

// identifyCriticalFindings extracts critical findings
func identifyCriticalFindings(categories []SecurityCategory) []CriticalFinding {
	var critical []CriticalFinding

	for _, cat := range categories {
		if cat.RiskLevel == RiskCritical {
			for _, file := range cat.Files {
				critical = append(critical, CriticalFinding{
					ID:          fmt.Sprintf("CF_%d", len(critical)+1),
					Title:       fmt.Sprintf("Critical Finding in %s", cat.Name),
					Description: fmt.Sprintf("Critical security issue detected: %s", strings.Join(file.RiskFactors, ", ")),
					Impact:      "High - Potential data breach or compliance violation",
					Location:    file.Path,
					Discovered:  file.ModifiedDate,
					Urgency:     RiskCritical,
				})
			}
		}
	}

	return critical
}

// generateRecommendations creates security recommendations
func generateRecommendations(categories []SecurityCategory) []Recommendation {
	var recommendations []Recommendation

	// General recommendations
	recommendations = append(recommendations, Recommendation{
		ID:          "REC_GEN_001",
		Title:       "Реализовать регулярное расписание стирания данных",
		Description: "Установить регулярное расписание операций стирания данных для обеспечения соответствия и минимизации риска утечки данных",
		Priority:    RiskMedium,
		Category:    "Общие",
		Effort:      "Средний",
		Deadline:    "30 дней",
	})

	// Category-specific recommendations
	for _, cat := range categories {
		if len(cat.Files) > 0 {
			rec := Recommendation{
				ID:          fmt.Sprintf("REC_%s_001", strings.ToUpper(strings.ReplaceAll(cat.Name, " ", "_"))),
				Title:       fmt.Sprintf("Устранить проблемы с %s", cat.Name),
				Description: fmt.Sprintf("Реализовать специфические меры для обработки %s: %s", cat.Name, strings.Join(cat.RiskFactors, ", ")),
				Priority:    cat.RiskLevel,
				Category:    cat.Name,
				Effort:      "Высокий",
				Deadline:    "14 дней",
			}
			recommendations = append(recommendations, rec)
		}
	}

	return recommendations
}

// Helper functions
func getScanMode(dryRun bool, profile string) string {
	if dryRun {
		return "Dry Run"
	}
	if profile != "" {
		return fmt.Sprintf("Profile: %s", profile)
	}
	return "Standard"
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func getSizeInBytes(sizeStr string) uint64 {
	// Simple parsing - in real implementation would be more robust
	if strings.HasSuffix(sizeStr, "MB") {
		var mb float64
		fmt.Sscanf(sizeStr, "%f", &mb)
		return uint64(mb * 1024 * 1024)
	} else if strings.HasSuffix(sizeStr, "GB") {
		var gb float64
		fmt.Sscanf(sizeStr, "%f", &gb)
		return uint64(gb * 1024 * 1024 * 1024)
	}
	return 0
}

func generateCategoryRecommendations(cat SecurityCategory) []string {
	switch cat.Name {
	case "Остатки данных":
		return []string{
			"Проверить завершение операций стирания",
			"Реализовать многопроходное стирание для конфиденциальных данных",
			"Использовать сертифицированные методы уничтожения данных",
		}
	case "Системные артефакты":
		return []string{
			"Очистить системные кэши и временные файлы",
			"Удалить следы активности пользователей",
			"Защитить файлы конфигурации системы",
		}
	case "Временные файлы":
		return []string{
			"Реализовать автоматическую очистку временных файлов",
			"Защитить временные директории приложений",
			"Мониторить создание временных файлов",
		}
	case "Системная очистка":
		return []string{
			"Регулярная очистка очереди печати",
			"Периодическая очистка DNS кэша",
			"Автоматическая очистка кэша браузеров",
			"Удаление старых системных логов",
		}
	default:
		return []string{
			"Регулярный мониторинг и очистка",
			"Реализовать автоматические проверки безопасности",
			"Документировать и проверять находки",
		}
	}
}

// SaveEnterpriseReport saves the enterprise report in specified format
func SaveEnterpriseReport(report *EnterpriseReport, cfg *config.Config, format string) error {
	if !cfg.Reporting.Enabled {
		return nil
	}

	// Create reports directory
	if err := os.MkdirAll(cfg.Reporting.LocalPath, 0755); err != nil {
		return fmt.Errorf("error creating reports directory: %w", err)
	}

	// Generate filename
	filename := fmt.Sprintf("wipedisk_security_audit_%s.%s",
		report.Metadata.ScanDate.Format("20060102_150405"), format)
	filepath := filepath.Join(cfg.Reporting.LocalPath, filename)

	switch format {
	case "json":
		return saveEnterpriseReportJSON(report, filepath)
	case "txt":
		return saveEnterpriseReportText(report, filepath)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// saveEnterpriseReportJSON saves report as JSON
func saveEnterpriseReportJSON(report *EnterpriseReport, filepath string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling report: %w", err)
	}

	return os.WriteFile(filepath, data, 0644)
}

// saveEnterpriseReportText saves report as formatted text
func saveEnterpriseReportText(report *EnterpriseReport, filepath string) error {
	var content strings.Builder

	// Header
	content.WriteString(fmt.Sprintf("%s - %s\n", report.Metadata.ProductName, report.Metadata.ReportTitle))
	content.WriteString(fmt.Sprintf("Сгенерировано: %s\n", report.GeneratedAt.Format("2006-01-02 15:04:05")))
	content.WriteString(fmt.Sprintf("Имя хоста: %s\n", report.Metadata.Hostname))
	content.WriteString(fmt.Sprintf("Дата сканирования: %s\n", report.Metadata.ScanDate.Format("2006-01-02 15:04:05")))
	content.WriteString(fmt.Sprintf("Режим сканирования: %s\n", report.Metadata.ScanMode))
	content.WriteString(fmt.Sprintf("Версия: %s\n", report.Metadata.Version))
	content.WriteString(fmt.Sprintf("Общий риск: %s\n", report.Metadata.OverallRisk))
	content.WriteString(fmt.Sprintf("ID запуска: %s\n", report.Metadata.RunID))
	content.WriteString(fmt.Sprintf("Длительность: %s\n", report.Metadata.Duration))
	content.WriteString(strings.Repeat("=", 80) + "\n\n")

	// Executive Summary
	content.WriteString("ИСПОЛНИТЕЛЬНАЯ СВОДКА\n")
	content.WriteString(strings.Repeat("-", 50) + "\n")
	content.WriteString(fmt.Sprintf("Всего находок: %d\n", report.Executive.TotalFindings))
	content.WriteString(fmt.Sprintf("Позиция безопасности: %s\n", report.Executive.SecurityPosture))
	content.WriteString(fmt.Sprintf("Риск соответствия: %s\n", report.Executive.ComplianceRisk))
	content.WriteString(fmt.Sprintf("Проанализировано дисков: %d\n", report.Executive.AnalyzedDisks))
	content.WriteString(fmt.Sprintf("Всего стерто данных: %s\n", report.Executive.TotalDataWiped))
	content.WriteString(fmt.Sprintf("Успешность: %.2f%%\n", report.Executive.SuccessRate))
	content.WriteString("\nРаспределение рисков:\n")
	for level, count := range report.Executive.RiskDistribution {
		content.WriteString(fmt.Sprintf("  %s: %d\n", level, count))
	}
	content.WriteString("\n")

	// Summary Table
	content.WriteString("СВОДНАЯ ТАБЛИЦА\n")
	content.WriteString(strings.Repeat("-", 50) + "\n")
	for _, cat := range report.SummaryTable.Categories {
		content.WriteString(fmt.Sprintf("%-20s %3d элементов  %s\n", cat.CategoryName, cat.FindingsCount, cat.RiskLevel))
	}
	content.WriteString("\n")

	// Critical Findings
	if len(report.Critical) > 0 {
		content.WriteString("КРИТИЧЕСКИЕ НАХОДКИ ТРЕБУЮЩИЕ НЕМЕДЛЕННОГО ВНИМАНИЯ\n")
		content.WriteString(strings.Repeat("-", 60) + "\n")
		for _, finding := range report.Critical {
			content.WriteString(fmt.Sprintf("[%s] %s\n", finding.ID, finding.Title))
			content.WriteString(fmt.Sprintf("  Расположение: %s\n", finding.Location))
			content.WriteString(fmt.Sprintf("  Влияние: %s\n", finding.Impact))
			content.WriteString(fmt.Sprintf("  Обнаружено: %s\n", finding.Discovered.Format("2006-01-02 15:04:05")))
			content.WriteString("\n")
		}
	}

	// Detailed Sections
	for _, section := range report.Sections {
		content.WriteString(fmt.Sprintf("%s\n", strings.ToUpper(section.CategoryName)))
		content.WriteString(strings.Repeat("-", len(section.CategoryName)) + "\n")
		content.WriteString(fmt.Sprintf("Описание: %s\n", section.Description))
		content.WriteString(fmt.Sprintf("Риск: %s\n", section.RiskLevel))
		content.WriteString(fmt.Sprintf("Всего файлов: %d\n", section.TotalFiles))
		content.WriteString(fmt.Sprintf("Факторы риска: %s\n", section.RiskExplanation))

		if len(section.TopExamples) > 0 {
			content.WriteString("\nТоп примеры:\n")
			for i, example := range section.TopExamples {
				content.WriteString(fmt.Sprintf("  %d. %s\n", i+1, example.Path))
				content.WriteString(fmt.Sprintf("     Размер: %s, Изменен: %s\n", example.Size, example.ModifiedDate.Format("2006-01-02")))
				content.WriteString(fmt.Sprintf("     Факторы риска: %s\n", strings.Join(example.RiskFactors, ", ")))
			}
		}
		content.WriteString("\n")
	}

	// Recommendations
	content.WriteString("РЕКОМЕНДАЦИИ ПО БЕЗОПАСНОСТИ\n")
	content.WriteString(strings.Repeat("-", 50) + "\n")
	for _, rec := range report.Recommendations {
		content.WriteString(fmt.Sprintf("[%s] %s\n", rec.ID, rec.Title))
		content.WriteString(fmt.Sprintf("  Приоритет: %s, Усилия: %s, Срок: %s\n", rec.Priority, rec.Effort, rec.Deadline))
		content.WriteString(fmt.Sprintf("  %s\n", rec.Description))
		content.WriteString("\n")
	}

	return os.WriteFile(filepath, []byte(content.String()), 0644)
}
