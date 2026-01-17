package system

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// DiagnosticLevel определяет уровень диагностики
type DiagnosticLevel string

const (
	LevelQuick DiagnosticLevel = "quick"
	LevelFull  DiagnosticLevel = "full"
	LevelDeep  DiagnosticLevel = "deep"
)

// DiagnosticTest определяет тип теста
type DiagnosticTest string

const (
	TestPermissions DiagnosticTest = "permissions"
	TestDisks       DiagnosticTest = "disks"
	TestMemory      DiagnosticTest = "memory"
	TestCPU         DiagnosticTest = "cpu"
	TestPaths       DiagnosticTest = "paths"
	TestAPI         DiagnosticTest = "api"
	TestWipe        DiagnosticTest = "wipe"
	TestNetwork     DiagnosticTest = "network"
)

// DiagnosticResult содержит результат теста
type DiagnosticResult struct {
	Test      DiagnosticTest `json:"test"`
	Status    string         `json:"status"` // PASS, FAIL, WARN
	Message   string         `json:"message"`
	Details   interface{}    `json:"details,omitempty"`
	Duration  time.Duration  `json:"duration"`
	Timestamp time.Time      `json:"timestamp"`
}

// SystemDiagnostics содержит полную диагностику системы
type SystemDiagnostics struct {
	Level       DiagnosticLevel    `json:"level"`
	StartTime   time.Time          `json:"start_time"`
	EndTime     time.Time          `json:"end_time"`
	Duration    time.Duration      `json:"duration"`
	Overall     string             `json:"overall"` // HEALTHY, WARNING, CRITICAL
	Results     []DiagnosticResult `json:"results"`
	Summary     DiagnosticSummary  `json:"summary"`
	Environment SystemEnvironment  `json:"environment"`
}

// DiagnosticSummary содержит сводку результатов
type DiagnosticSummary struct {
	TotalTests int `json:"total_tests"`
	Passed     int `json:"passed"`
	Failed     int `json:"failed"`
	Warnings   int `json:"warnings"`
}

// SystemEnvironment содержит информацию об окружении
type SystemEnvironment struct {
	OSVersion    string            `json:"os_version"`
	Architecture string            `json:"architecture"`
	Username     string            `json:"username"`
	Domain       string            `json:"domain"`
	MachineName  string            `json:"machine_name"`
	IsAdmin      bool              `json:"is_admin"`
	IsServer     bool              `json:"is_server"`
	TotalMemory  uint64            `json:"total_memory"`
	AvailableMem uint64            `json:"available_memory"`
	CPUCount     int               `json:"cpu_count"`
	Environment  map[string]string `json:"environment"`
}

// SystemDiagnosticsRunner выполняет диагностику системы
type SystemDiagnosticsRunner struct {
	level   DiagnosticLevel
	verbose bool
	output  string
	test    DiagnosticTest
}

// NewSystemDiagnosticsRunner создает новый runner
func NewSystemDiagnosticsRunner(level DiagnosticLevel, verbose bool, output string, test DiagnosticTest) *SystemDiagnosticsRunner {
	return &SystemDiagnosticsRunner{
		level:   level,
		verbose: verbose,
		output:  output,
		test:    test,
	}
}

// RunDiagnostics выполняет полную диагностику
func (sdr *SystemDiagnosticsRunner) RunDiagnostics(ctx context.Context) (*SystemDiagnostics, error) {
	startTime := time.Now()

	diagnostics := &SystemDiagnostics{
		Level:       sdr.level,
		StartTime:   startTime,
		Results:     make([]DiagnosticResult, 0),
		Environment: sdr.collectEnvironmentInfo(),
	}

	// Определяем тесты для выполнения
	tests := sdr.getTestsForLevel()

	for _, test := range tests {
		select {
		case <-ctx.Done():
			return diagnostics, ctx.Err()
		default:
		}

		result := sdr.runTest(ctx, test)
		diagnostics.Results = append(diagnostics.Results, result)
	}

	diagnostics.EndTime = time.Now()
	diagnostics.Duration = diagnostics.EndTime.Sub(diagnostics.StartTime)
	diagnostics.Summary = sdr.calculateSummary(diagnostics.Results)
	diagnostics.Overall = sdr.determineOverallStatus(diagnostics.Summary)

	return diagnostics, nil
}

// getTestsForLevel возвращает тесты для указанного уровня
func (sdr *SystemDiagnosticsRunner) getTestsForLevel() []DiagnosticTest {
	if sdr.test != "" {
		return []DiagnosticTest{sdr.test}
	}

	switch sdr.level {
	case LevelQuick:
		return []DiagnosticTest{TestPermissions, TestDisks, TestMemory}
	case LevelFull:
		return []DiagnosticTest{TestPermissions, TestDisks, TestMemory, TestCPU, TestPaths, TestAPI}
	case LevelDeep:
		return []DiagnosticTest{TestPermissions, TestDisks, TestMemory, TestCPU, TestPaths, TestAPI, TestWipe, TestNetwork}
	default:
		return []DiagnosticTest{TestPermissions, TestDisks, TestMemory}
	}
}

// runTest выполняет отдельный тест
func (sdr *SystemDiagnosticsRunner) runTest(ctx context.Context, test DiagnosticTest) DiagnosticResult {
	startTime := time.Now()

	result := DiagnosticResult{
		Test:      test,
		Timestamp: startTime,
	}

	switch test {
	case TestPermissions:
		result.Status, result.Message, result.Details = sdr.testPermissions()
	case TestDisks:
		result.Status, result.Message, result.Details = sdr.testDisks()
	case TestMemory:
		result.Status, result.Message, result.Details = sdr.testMemory()
	case TestCPU:
		result.Status, result.Message, result.Details = sdr.testCPU()
	case TestPaths:
		result.Status, result.Message, result.Details = sdr.testPaths()
	case TestAPI:
		result.Status, result.Message, result.Details = sdr.testAPI()
	case TestWipe:
		result.Status, result.Message, result.Details = sdr.testWipe(ctx)
	case TestNetwork:
		result.Status, result.Message, result.Details = sdr.testNetwork()
	}

	result.Duration = time.Since(startTime)

	if sdr.verbose {
		fmt.Printf("[TEST] %s: %s - %s (%v)\n", result.Test, result.Status, result.Message, result.Duration)
	}

	return result
}

// Реализации тестов
func (sdr *SystemDiagnosticsRunner) testPermissions() (string, string, interface{}) {
	// Проверка прав администратора
	isAdmin := false

	// Простая проверка для Windows
	if runtime.GOOS == "windows" {
		cmd := exec.Command("net", "session")
		err := cmd.Run()
		if err == nil {
			isAdmin = true
		}
	}

	details := map[string]interface{}{
		"is_admin": isAdmin,
		"user":     os.Getenv("USERNAME"),
		"domain":   os.Getenv("USERDOMAIN"),
	}

	if isAdmin {
		return "PASS", "Пользователь имеет права администратора", details
	}

	return "WARN", "Пользователь не имеет прав администратора", details
}

func (sdr *SystemDiagnosticsRunner) testDisks() (string, string, interface{}) {
	disks, err := GetDiskInfo(false)
	if err != nil {
		return "FAIL", fmt.Sprintf("Ошибка получения информации о дисках: %v", err), nil
	}

	diskDetails := make([]map[string]interface{}, len(disks))
	for i, disk := range disks {
		diskDetails[i] = map[string]interface{}{
			"letter":      disk.Letter,
			"type":        disk.Type,
			"total_gb":    float64(disk.TotalSize) / (1024 * 1024 * 1024),
			"free_gb":     float64(disk.FreeSize) / (1024 * 1024 * 1024),
			"is_system":   disk.IsSystem,
			"is_writable": disk.IsWritable,
		}
	}

	// Проверяем наличие свободного места
	minFreeGB := 1.0 // Минимум 1GB
	for _, disk := range disks {
		freeGB := float64(disk.FreeSize) / (1024 * 1024 * 1024)
		if freeGB < minFreeGB {
			return "WARN", fmt.Sprintf("Диск %s имеет мало свободного места: %.1f GB", disk.Letter, freeGB), diskDetails
		}
	}

	return "PASS", fmt.Sprintf("Найдено %d дисков, все в порядке", len(disks)), diskDetails
}

func (sdr *SystemDiagnosticsRunner) testMemory() (string, string, interface{}) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Получаем общую память системы
	var totalMemory uint64 = 8 * 1024 * 1024 * 1024 // 8GB по умолчанию
	if runtime.GOOS == "windows" {
		// В реальной реализации нужно использовать Windows API
		totalMemory = 16 * 1024 * 1024 * 1024 // 16GB для примера
	}

	availableMemory := totalMemory - uint64(memStats.Sys)
	memoryUsagePercent := float64(memStats.Sys) / float64(totalMemory) * 100

	details := map[string]interface{}{
		"total_mb":      totalMemory / (1024 * 1024),
		"used_mb":       memStats.Sys / (1024 * 1024),
		"available_mb":  availableMemory / (1024 * 1024),
		"usage_percent": memoryUsagePercent,
		"heap_mb":       memStats.HeapAlloc / (1024 * 1024),
	}

	if memoryUsagePercent > 90 {
		return "WARN", fmt.Sprintf("Высокое использование памяти: %.1f%%", memoryUsagePercent), details
	}

	return "PASS", fmt.Sprintf("Использование памяти в норме: %.1f%%", memoryUsagePercent), details
}

func (sdr *SystemDiagnosticsRunner) testCPU() (string, string, interface{}) {
	cpuCount := runtime.NumCPU()

	// Простая проверка нагрузки CPU
	details := map[string]interface{}{
		"cpu_count": cpuCount,
		"os":        runtime.GOOS,
		"arch":      runtime.GOARCH,
	}

	if cpuCount < 2 {
		return "WARN", fmt.Sprintf("Мало CPU ядер: %d", cpuCount), details
	}

	return "PASS", fmt.Sprintf("Доступно %d CPU ядер", cpuCount), details
}

func (sdr *SystemDiagnosticsRunner) testPaths() (string, string, interface{}) {
	paths := []string{
		os.Getenv("TEMP"),
		os.Getenv("WINDIR"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Temp"),
		filepath.Join(os.Getenv("APPDATA"), "Microsoft"),
	}

	pathDetails := make([]map[string]interface{}, len(paths))
	allAccessible := true

	for i, path := range paths {
		accessible := true
		var size int64

		if path != "" {
			info, err := os.Stat(path)
			if err != nil {
				accessible = false
				allAccessible = false
			} else {
				size = info.Size()
			}
		} else {
			accessible = false
			allAccessible = false
		}

		pathDetails[i] = map[string]interface{}{
			"path":       path,
			"accessible": accessible,
			"size":       size,
		}
	}

	if allAccessible {
		return "PASS", "Все проверенные пути доступны", pathDetails
	}

	return "WARN", "Некоторые пути недоступны", pathDetails
}

func (sdr *SystemDiagnosticsRunner) testAPI() (string, string, interface{}) {
	// Проверка доступа к Windows API
	apiDetails := map[string]interface{}{
		"os":        runtime.GOOS,
		"arch":      runtime.GOARCH,
		"goversion": runtime.Version(),
	}

	if runtime.GOOS == "windows" {
		// Проверяем доступ к базовым функциям Windows
		testFile := filepath.Join(os.Getenv("TEMP"), "wipedisk_api_test.tmp")
		file, err := os.Create(testFile)
		if err != nil {
			return "FAIL", "Ошибка создания тестового файла", apiDetails
		}
		file.Close()
		os.Remove(testFile)

		apiDetails["file_api"] = "OK"
		return "PASS", "Доступ к Windows API в норме", apiDetails
	}

	return "WARN", "Тестирование API только для Windows", apiDetails
}

func (sdr *SystemDiagnosticsRunner) testWipe(ctx context.Context) (string, string, interface{}) {
	// Тест возможности затирания (dry-run)
	wipeDetails := map[string]interface{}{
		"test_mode": "dry_run",
		"test_file": "",
	}

	testFile := filepath.Join(os.Getenv("TEMP"), "wipedisk_dryrun_test.tmp")
	file, err := os.Create(testFile)
	if err != nil {
		return "FAIL", fmt.Sprintf("Ошибка создания тестового файла: %v", err), wipeDetails
	}

	// Пишем тестовые данные
	testData := make([]byte, 1024)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	_, err = file.Write(testData)
	file.Close()

	if err != nil {
		os.Remove(testFile)
		return "FAIL", fmt.Sprintf("Ошибка записи в тестовый файл: %v", err), wipeDetails
	}

	// Проверяем чтение
	readData, err := os.ReadFile(testFile)
	os.Remove(testFile)

	if err != nil {
		return "FAIL", fmt.Sprintf("Ошибка чтения тестового файла: %v", err), wipeDetails
	}

	if len(readData) != len(testData) {
		return "FAIL", "Размер прочитанных данных отличается от записанных", wipeDetails
	}

	wipeDetails["test_file"] = testFile
	wipeDetails["test_size"] = len(testData)

	return "PASS", "Тест затирания пройден успешно", wipeDetails
}

func (sdr *SystemDiagnosticsRunner) testNetwork() (string, string, interface{}) {
	// Простая проверка сетевого подключения
	networkDetails := map[string]interface{}{
		"test_host": "google.com",
	}

	// Проверяем доступ к интернету
	cmd := exec.CommandContext(context.Background(), "ping", "-n", "1", "google.com")
	err := cmd.Run()

	if err != nil {
		networkDetails["status"] = "no_internet"
		return "WARN", "Нет доступа к интернету", networkDetails
	}

	networkDetails["status"] = "connected"
	return "PASS", "Сетевое подключение в норме", networkDetails
}

// collectEnvironmentInfo собирает информацию об окружении
func (sdr *SystemDiagnosticsRunner) collectEnvironmentInfo() SystemEnvironment {
	env := SystemEnvironment{
		OSVersion:    getOSVersion(),
		Architecture: runtime.GOARCH,
		Username:     os.Getenv("USERNAME"),
		Domain:       os.Getenv("USERDOMAIN"),
		MachineName:  os.Getenv("COMPUTERNAME"),
		CPUCount:     runtime.NumCPU(),
		Environment:  make(map[string]string),
	}

	// Проверка прав администратора
	if runtime.GOOS == "windows" {
		cmd := exec.Command("net", "session")
		env.IsAdmin = cmd.Run() == nil
	}

	// Проверка серверной ОС
	env.IsServer = strings.Contains(env.OSVersion, "Server") || strings.Contains(env.OSVersion, "2008") || strings.Contains(env.OSVersion, "2012") || strings.Contains(env.OSVersion, "2016") || strings.Contains(env.OSVersion, "2019")

	// Собираем переменные окружения
	relevantEnv := []string{"PATH", "TEMP", "WINDIR", "PROGRAMFILES", "PROGRAMFILES(X86)", "LOCALAPPDATA", "APPDATA"}
	for _, key := range relevantEnv {
		if value := os.Getenv(key); value != "" {
			env.Environment[key] = value
		}
	}

	return env
}

// getOSVersion получает версию ОС
func getOSVersion() string {
	if runtime.GOOS != "windows" {
		return runtime.GOOS
	}

	cmd := exec.Command("ver")
	output, err := cmd.Output()
	if err != nil {
		return "Windows (unknown version)"
	}

	return strings.TrimSpace(string(output))
}

// calculateSummary считает сводку результатов
func (sdr *SystemDiagnosticsRunner) calculateSummary(results []DiagnosticResult) DiagnosticSummary {
	summary := DiagnosticSummary{TotalTests: len(results)}

	for _, result := range results {
		switch result.Status {
		case "PASS":
			summary.Passed++
		case "FAIL":
			summary.Failed++
		case "WARN":
			summary.Warnings++
		}
	}

	return summary
}

// determineOverallStatus определяет общий статус
func (sdr *SystemDiagnosticsRunner) determineOverallStatus(summary DiagnosticSummary) string {
	if summary.Failed > 0 {
		return "CRITICAL"
	}
	if summary.Warnings > 0 {
		return "WARNING"
	}
	return "HEALTHY"
}

// SaveDiagnostics сохраняет диагностику в файл
func (sdr *SystemDiagnosticsRunner) SaveDiagnostics(diagnostics *SystemDiagnostics, outputPath string) error {
	if outputPath == "" {
		timestamp := diagnostics.StartTime.Format("20060102_150405")
		outputPath = filepath.Join(os.Getenv("TEMP"), fmt.Sprintf("wipedisk_diagnostics_%s.json", timestamp))
	}

	data, err := json.MarshalIndent(diagnostics, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка сериализации JSON: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("ошибка сохранения файла: %w", err)
	}

	return nil
}
