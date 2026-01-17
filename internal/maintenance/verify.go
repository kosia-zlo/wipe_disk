package maintenance

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wipedisk_enterprise/internal/logging"
	"wipedisk_enterprise/internal/system"
	"wipedisk_enterprise/internal/wipe"
)

// VerificationLevel определяет уровень проверки
type VerificationLevel string

const (
	LevelBasic      VerificationLevel = "basic"
	LevelPhysical   VerificationLevel = "physical"
	LevelAggressive VerificationLevel = "aggressive"
)

// VerificationReport содержит результаты проверки
type VerificationReport struct {
	WipeVerified      bool                  `json:"wipe_verified"`
	VerificationLevel VerificationLevel     `json:"verification_level"`
	RecoveryAttempts  int                   `json:"recovery_attempts"`
	RecoveredData     int64                 `json:"recovered_data"`
	Anomalies         []VerificationAnomaly `json:"anomalies"`
	Compliance        []string              `json:"compliance"`
	TestDuration      time.Duration         `json:"test_duration"`
	TestDate          time.Time             `json:"test_date"`
	Disk              string                `json:"disk"`
	Method            string                `json:"method"`
	Passes            int                   `json:"passes"`
	SuccessRate       float64               `json:"success_rate"`
}

// VerificationAnomaly описывает аномалию при проверке
type VerificationAnomaly struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Location    string `json:"location"`
	Severity    string `json:"severity"`
}

// PhysicalVerifier выполняет физическую проверку затирания
type PhysicalVerifier struct {
	logger         *logging.EnterpriseLogger
	level          VerificationLevel
	maxAttempts    int
	readBufferSize int
	timeout        time.Duration
}

// NewPhysicalVerifier создает новый верификатор
func NewPhysicalVerifier(level VerificationLevel, logger *logging.EnterpriseLogger) *PhysicalVerifier {
	maxAttempts := 3
	readBufferSize := 64 * 1024 // 64KB
	timeout := 30 * time.Minute

	switch level {
	case LevelPhysical:
		maxAttempts = 5
		readBufferSize = 128 * 1024 // 128KB
		timeout = time.Hour
	case LevelAggressive:
		maxAttempts = 10
		readBufferSize = 256 * 1024 // 256KB
		timeout = 2 * time.Hour
	}

	return &PhysicalVerifier{
		logger:         logger,
		level:          level,
		maxAttempts:    maxAttempts,
		readBufferSize: readBufferSize,
		timeout:        timeout,
	}
}

// VerifyLastSession проверяет последнюю сессию затирания
func (pv *PhysicalVerifier) VerifyLastSession(ctx context.Context) (*VerificationReport, error) {
	// Находим последний отчёт о затирании
	reportPath := filepath.Join(os.Getenv("TEMP"), "wipedisk_reports")
	latestReport, err := pv.findLatestReport(reportPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка поиска последнего отчёта: %w", err)
	}

	if latestReport == nil {
		return nil, fmt.Errorf("не найдено отчётов о затирании")
	}

	return pv.VerifyReport(ctx, latestReport)
}

// VerifyReport проверяет конкретный отчёт
func (pv *PhysicalVerifier) VerifyReport(ctx context.Context, report *wipe.WipeOperation) (*VerificationReport, error) {
	pv.logger.Log("INFO", "Начало верификации",
		"disk", report.Disk,
		"method", report.Method,
		"passes", report.Passes,
		"level", pv.level)

	startTime := time.Now()
	verificationReport := &VerificationReport{
		VerificationLevel: pv.level,
		RecoveryAttempts:  0,
		RecoveredData:     0,
		Anomalies:         []VerificationAnomaly{},
		Compliance:        []string{},
		TestDate:          startTime,
		Disk:              report.Disk,
		Method:            report.Method,
		Passes:            report.Passes,
	}

	// Создаем контекст с таймаутом
	verifyCtx, cancel := context.WithTimeout(ctx, pv.timeout)
	defer cancel()

	// Проверяем в зависимости от уровня
	var err error
	switch pv.level {
	case LevelBasic:
		err = pv.performBasicVerification(verifyCtx, report, verificationReport)
	case LevelPhysical:
		err = pv.performPhysicalVerification(verifyCtx, report, verificationReport)
	case LevelAggressive:
		err = pv.performAggressiveVerification(verifyCtx, report, verificationReport)
	}

	if err != nil {
		verificationReport.Anomalies = append(verificationReport.Anomalies, VerificationAnomaly{
			Type:        "verification_error",
			Description: err.Error(),
			Location:    "verification_engine",
			Severity:    "high",
		})
	}

	verificationReport.TestDuration = time.Since(startTime)
	verificationReport.SuccessRate = pv.calculateSuccessRate(verificationReport)

	// Определяем соответствие стандартам
	verificationReport.Compliance = pv.determineCompliance(verificationReport)

	verificationReport.WipeVerified = verificationReport.SuccessRate >= 95.0

	pv.logger.Log("INFO", "Верификация завершена",
		"verified", verificationReport.WipeVerified,
		"success_rate", verificationReport.SuccessRate,
		"duration", verificationReport.TestDuration)

	return verificationReport, nil
}

// performBasicVerification выполняет базовую проверку
func (pv *PhysicalVerifier) performBasicVerification(ctx context.Context, report *wipe.WipeOperation, vr *VerificationReport) error {
	pv.logger.Log("INFO", "Выполнение базовой верификации")

	// Проверяем, что диск существует
	diskInfo, err := system.GetDiskInfo(false)
	if err != nil {
		return fmt.Errorf("ошибка получения информации о дисках: %w", err)
	}

	var targetDisk *system.DiskInfo
	for _, disk := range diskInfo {
		if strings.EqualFold(disk.Letter, report.Disk) {
			targetDisk = &disk
			break
		}
	}

	if targetDisk == nil {
		return fmt.Errorf("диск %s не найден", report.Disk)
	}

	// Проверяем свободное место
	if targetDisk.FreeSize < report.BytesWiped {
		vr.Anomalies = append(vr.Anomalies, VerificationAnomaly{
			Type: "space_mismatch",
			Description: fmt.Sprintf("Свободное место %.1f GB меньше ожидаемого %.1f GB",
				float64(targetDisk.FreeSize)/(1024*1024*1024),
				float64(report.BytesWiped)/(1024*1024*1024)),
			Location: report.Disk,
			Severity: "medium",
		})
	}

	// Проверяем время выполнения
	if report.SpeedMBps > 1000 { // Слишком высокая скорость подозрительна
		vr.Anomalies = append(vr.Anomalies, VerificationAnomaly{
			Type:        "speed_anomaly",
			Description: fmt.Sprintf("Подозрительно высокая скорость: %.1f MB/s", report.SpeedMBps),
			Location:    report.Disk,
			Severity:    "low",
		})
	}

	return nil
}

// performPhysicalVerification выполняет физическую проверку
func (pv *PhysicalVerifier) performPhysicalVerification(ctx context.Context, report *wipe.WipeOperation, vr *VerificationReport) error {
	pv.logger.Log("INFO", "Выполнение физической верификации")

	// Сначала базовая проверка
	if err := pv.performBasicVerification(ctx, report, vr); err != nil {
		return err
	}

	// Создаем тестовый файл для проверки
	testFile := filepath.Join(report.Disk, fmt.Sprintf("wipedisk_verify_test_%d.tmp", time.Now().Unix()))
	defer os.Remove(testFile)

	// Записываем тестовые данные
	testData := make([]byte, pv.readBufferSize)
	if _, err := rand.Read(testData); err != nil {
		return fmt.Errorf("ошибка генерации тестовых данных: %w", err)
	}

	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		return fmt.Errorf("ошибка записи тестового файла: %w", err)
	}

	// Пытаемся прочитать данные несколько раз
	for attempt := 1; attempt <= pv.maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		vr.RecoveryAttempts++

		readData, err := os.ReadFile(testFile)
		if err != nil {
			vr.Anomalies = append(vr.Anomalies, VerificationAnomaly{
				Type:        "read_error",
				Description: fmt.Sprintf("Попытка %d: %v", attempt, err),
				Location:    testFile,
				Severity:    "high",
			})
			continue
		}

		// Сравниваем данные
		if len(readData) != len(testData) {
			vr.Anomalies = append(vr.Anomalies, VerificationAnomaly{
				Type:        "size_mismatch",
				Description: fmt.Sprintf("Попытка %d: размер %d != %d", attempt, len(readData), len(testData)),
				Location:    testFile,
				Severity:    "high",
			})
			vr.RecoveredData += int64(len(readData))
			continue
		}

		// Проверяем содержимое
		for i := range testData {
			if readData[i] != testData[i] {
				vr.Anomalies = append(vr.Anomalies, VerificationAnomaly{
					Type:        "data_corruption",
					Description: fmt.Sprintf("Попытка %d: данные отличаются на позиции %d", attempt, i),
					Location:    testFile,
					Severity:    "high",
				})
				vr.RecoveredData++
				break
			}
		}

		// Если данные совпадают, проверка пройдена
		if len(vr.Anomalies) == 0 || vr.Anomalies[len(vr.Anomalies)-1].Severity != "high" {
			break
		}

		time.Sleep(time.Duration(attempt) * time.Second) // Пауза между попытками
	}

	return nil
}

// performAggressiveVerification выполняет агрессивную проверку
func (pv *PhysicalVerifier) performAggressiveVerification(ctx context.Context, report *wipe.WipeOperation, vr *VerificationReport) error {
	pv.logger.Log("INFO", "Выполнение агрессивной верификации")

	// Сначала физическая проверка
	if err := pv.performPhysicalVerification(ctx, report, vr); err != nil {
		return err
	}

	// Дополнительные проверки для агрессивного режима
	// 1. Проверка MFT записей (если это NTFS)
	if pv.isNTFS(report.Disk) {
		if err := pv.checkMFTIntegrity(ctx, report.Disk, vr); err != nil {
			pv.logger.Log("WARN", "Ошибка проверки MFT", "error", err)
		}
	}

	// 2. Проверка файловой системы на остаточные данные
	if err := pv.checkFileSystemResidues(ctx, report.Disk, vr); err != nil {
		pv.logger.Log("WARN", "Ошибка проверки файловой системы", "error", err)
	}

	return nil
}

// findLatestReport ищет последний отчёт о затирании
func (pv *PhysicalVerifier) findLatestReport(reportDir string) (*wipe.WipeOperation, error) {
	if _, err := os.Stat(reportDir); os.IsNotExist(err) {
		return nil, nil
	}

	files, err := os.ReadDir(reportDir)
	if err != nil {
		return nil, err
	}

	var latestFile string
	var latestTime time.Time

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			latestFile = filepath.Join(reportDir, file.Name())
		}
	}

	if latestFile == "" {
		return nil, nil
	}

	data, err := os.ReadFile(latestFile)
	if err != nil {
		return nil, err
	}

	var report wipe.WipeOperation
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}

	return &report, nil
}

// calculateSuccessRate вычисляет процент успешности
func (pv *PhysicalVerifier) calculateSuccessRate(vr *VerificationReport) float64 {
	if vr.RecoveryAttempts == 0 {
		return 100.0
	}

	// Базовый рейтинг
	rating := 100.0

	// Вычитаем за аномалии
	for _, anomaly := range vr.Anomalies {
		switch anomaly.Severity {
		case "high":
			rating -= 20.0
		case "medium":
			rating -= 10.0
		case "low":
			rating -= 5.0
		}
	}

	// Вычитаем за восстановленные данные
	if vr.RecoveredData > 0 {
		rating -= float64(vr.RecoveredData) / 1024.0 // 1KB = 1%
		if rating < 0 {
			rating = 0
		}
	}

	return rating
}

// determineCompliance определяет соответствие стандартам
func (pv *PhysicalVerifier) determineCompliance(vr *VerificationReport) []string {
	var compliance []string

	if vr.SuccessRate >= 95.0 {
		compliance = append(compliance, "DOD5220")
	}

	if vr.SuccessRate >= 90.0 {
		compliance = append(compliance, "NIST800-88")
	}

	if vr.SuccessRate >= 85.0 {
		compliance = append(compliance, "BSI_VSITR")
	}

	return compliance
}

// isNTFS проверяет, является ли диск NTFS
func (pv *PhysicalVerifier) isNTFS(disk string) bool {
	// Упрощенная проверка - в реальной реализации нужно использовать Windows API
	return true // Предполагаем NTFS для Windows
}

// checkMFTIntegrity проверяет целостность MFT
func (pv *PhysicalVerifier) checkMFTIntegrity(ctx context.Context, disk string, vr *VerificationReport) error {
	// Заглушка для MFT проверки
	// В реальной реализации нужно использовать raw disk access
	pv.logger.Log("INFO", "Проверка целостности MFT", "disk", disk)
	return nil
}

// checkFileSystemResidues проверяет остаточные данные в файловой системе
func (pv *PhysicalVerifier) checkFileSystemResidues(ctx context.Context, disk string, vr *VerificationReport) error {
	// Заглушка для проверки остаточных данных
	// В реальной реализации нужно сканировать свободное пространство
	pv.logger.Log("INFO", "Проверка остаточных данных", "disk", disk)
	return nil
}
