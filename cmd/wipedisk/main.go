package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"wipedisk_enterprise/internal/cli"
	"wipedisk_enterprise/internal/config"
	"wipedisk_enterprise/internal/logging"
	"wipedisk_enterprise/internal/maintenance"
	"wipedisk_enterprise/internal/reporting"
	"wipedisk_enterprise/internal/security"
	"wipedisk_enterprise/internal/system"
	"wipedisk_enterprise/internal/wipe"
)

const (
	Version = "1.2.1.1"
	AppName = "WipeDisk Enterprise"

	// Exit codes
	EXIT_SUCCESS = 0
	EXIT_WARNING = 2
	EXIT_ERROR   = 1
)

var (
	cfg             *config.Config
	logger          *logging.EnterpriseLogger
	dryRun          bool
	verbose         bool
	configPath      string
	maxDuration     time.Duration
	maxDurationStr  string
	profile         string
	engine          string
	mode            string
	allowSystemDisk bool
	startTime       time.Time
)

// CLI команды
var rootCmd = &cobra.Command{
	Use:     "wipedisk",
	Short:   "WipeDisk Enterprise - утилита для очистки дисков",
	Long:    "Enterprise утилита для безопасной очистки дисков и затирания свободного места",
	Version: Version,
}

var wipeCmd = &cobra.Command{
	Use:   "wipe [диски]",
	Short: "Затереть свободное место на дисках",
	RunE:  runWipe,
}

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Очистить временные файлы",
	RunE:  runClean,
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Показать информацию о дисках",
	RunE:  runInfo,
}

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Проверить качество затирания",
	RunE:  runVerify,
}

var maintenanceCmd = &cobra.Command{
	Use:   "maintenance",
	Short: "Единый режим обслуживания",
	RunE:  runMaintenance,
}

var diagnoseCmd = &cobra.Command{
	Use:   "diagnose",
	Short: "Самодиагностика системы",
	RunE:  runDiagnose,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "n", false, "Тестовый режим")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Подробный вывод")
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Путь к конфигурации")
	rootCmd.PersistentFlags().StringVar(&maxDurationStr, "max-duration", "", "Максимальное время работы (например: 30m, 2h)")
	rootCmd.PersistentFlags().StringVar(&profile, "profile", "", "Профиль производительности (safe/balanced/aggressive/fast/sdelete)")
	rootCmd.PersistentFlags().StringVar(&engine, "engine", "internal", "Движок затирания (internal/sdelete-compatible/cipher)")
	rootCmd.PersistentFlags().StringVar(&mode, "mode", "standard", "Режим затирания (standard/sdelete/cipher)")
	rootCmd.PersistentFlags().BoolVar(&allowSystemDisk, "allow-system-disk", false, "Разрешить затирание системного диска (ОПАСНО)")

	wipeCmd.Flags().StringP("method", "m", "", "Метод затирания")
	wipeCmd.Flags().IntP("passes", "p", 0, "Количество проходов")
	wipeCmd.Flags().BoolP("force", "f", false, "Пропустить подтверждение")

	verifyCmd.Flags().Bool("last-session", false, "Проверить последнюю сессию")
	verifyCmd.Flags().Bool("physical", false, "Физическая проверка (требует админ)")
	verifyCmd.Flags().String("report", "", "Сохранить отчёт в файл")
	verifyCmd.Flags().String("format", "json", "Формат отчёта (json/csv)")
	verifyCmd.Flags().String("level", "basic", "Уровень проверки (basic/physical/aggressive)")

	maintenanceCmd.Flags().String("plan", "", "План обслуживания (full_year/light_monthly/security_quarterly/quick_cleanup/deep_clean/verify_only)")
	maintenanceCmd.Flags().Bool("list-plans", false, "Показать доступные планы")
	maintenanceCmd.Flags().Bool("silent", false, "Тихий режим")
	maintenanceCmd.Flags().Bool("parallel", false, "Параллельное выполнение")

	diagnoseCmd.Flags().Bool("quick", false, "Быстрая диагностика")
	diagnoseCmd.Flags().Bool("full", false, "Полная диагностика")
	diagnoseCmd.Flags().Bool("deep", false, "Глубокая диагностика")
	diagnoseCmd.Flags().String("test", "", "Конкретный тест (permissions/disks/memory/cpu/paths/api/wipe/network)")
	diagnoseCmd.Flags().String("output", "", "Сохранить отчёт в файл")

	// Cleanup command
	cleanupCmd := &cobra.Command{
		Use:   "cleanup [operations...]",
		Short: "Выполнить операции очистки системы",
		Long:  "Выполнение операций очистки системы: очереди печати, DNS кэш, кэш браузеров, временные файлы, старые логи",
		RunE:  runCleanup,
	}
	cleanupCmd.Flags().Bool("list", false, "Показать доступные операции очистки")
	cleanupCmd.Flags().String("category", "", "Выполнить операции по категории")
	cleanupCmd.Flags().Bool("dry-run", false, "Тестовый режим выполнения")

	rootCmd.AddCommand(wipeCmd, cleanCmd, infoCmd, verifyCmd, maintenanceCmd, diagnoseCmd, cleanupCmd)
}

func runWipe(cmd *cobra.Command, args []string) error {
	startTime = time.Now()

	if err := security.SecurityChecks(config.Default()); err != nil {
		return err
	}

	// СНАЧАЛА загружаем конфигурацию
	var err error
	cfg, err = config.Load(configPath)
	if err != nil {
		return fmt.Errorf("ошибка загрузки конфигурации: %w", err)
	}

	// Валидация конфигурации
	if err := config.Validate(cfg); err != nil {
		return fmt.Errorf("невалидная конфигурация: %w", err)
	}

	// Применяем профиль если указан
	if profile != "" {
		if err := config.ApplyProfile(cfg, profile); err != nil {
			return fmt.Errorf("ошибка применения профиля %s: %w", profile, err)
		}
	}

	// Создаем логгер после загрузки конфигурации
	logger, err = logging.NewEnterpriseLogger(cfg, verbose)
	if err != nil {
		return fmt.Errorf("ошибка инициализации логгера: %w", err)
	}
	defer logger.Close()

	if profile != "" {
		logger.Log("INFO", "Применён профиль", "profile", profile)
	}

	// Валидация режима
	validMode, err := wipe.ValidateMode(mode)
	if err != nil {
		return fmt.Errorf("некорректный режим: %w", err)
	}

	// Парсинг max-duration (после загрузки конфига)
	if maxDurationStr != "" {
		duration, err := time.ParseDuration(maxDurationStr)
		if err != nil {
			return fmt.Errorf("неверный формат max-duration: %w", err)
		}
		maxDuration = duration
	} else if cfg.Wipe.MaxDuration != "" {
		duration, err := time.ParseDuration(cfg.Wipe.MaxDuration)
		if err != nil {
			return fmt.Errorf("неверный формат max_duration в конфиге: %w", err)
		}
		maxDuration = duration
	}

	// Автоматическое создание директорий для отчетов
	if cfg.Reporting.Enabled {
		if err := os.MkdirAll(cfg.Reporting.LocalPath, 0755); err != nil {
			return fmt.Errorf("ошибка создания директории для отчетов: %w", err)
		}
	}

	logger.Log("INFO", "Запуск WipeDisk Enterprise", "version", Version, "dry_run", dryRun)

	disks, err := system.GetDiskInfo(verbose)
	if err != nil {
		return fmt.Errorf("ошибка получения дисков: %w", err)
	}

	var targetDisks []system.DiskInfo
	if len(args) > 0 {
		// Process only specified disks
		for _, arg := range args {
			for _, disk := range disks {
				if strings.EqualFold(disk.Letter, arg) || strings.EqualFold(disk.Letter+":", arg) {
					if !security.ShouldSkipDisk(cfg, disk) {
						targetDisks = append(targetDisks, disk)
					}
					break
				}
			}
		}
	} else {
		// Process all non-excluded disks
		for _, disk := range disks {
			if !security.ShouldSkipDisk(cfg, disk) {
				targetDisks = append(targetDisks, disk)
			}
		}
	}

	if len(targetDisks) == 0 {
		logger.Log("WARN", "Нет доступных дисков для обработки")
		return nil
	}

	force, _ := cmd.Flags().GetBool("force")
	if !force && !dryRun && cfg.Security.RequireConfirmation {
		fmt.Printf("ВНИМАНИЕ: Будет затерто свободное место на %d локальных дисках:\n", len(targetDisks))
		for _, disk := range targetDisks {
			fmt.Printf("  %s (%s, %.1f GB свободно)\n", disk.Letter, disk.Type, float64(disk.FreeSize)/(1024*1024*1024))
		}
		fmt.Print("Продолжить? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			logger.Log("INFO", "Операция отменена пользователем")
			return nil
		}
	}

	// Создаем контекст с учетом maxDuration
	baseCtx := context.Background()
	var ctx context.Context
	var cancel context.CancelFunc

	if maxDuration > 0 {
		ctx, cancel = context.WithTimeout(baseCtx, maxDuration)
	} else {
		ctx, cancel = context.WithCancel(baseCtx)
	}
	defer cancel()

	// Установка обработчиков сигналов
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Горутина для обработки сигналов
	go func() {
		sig := <-sigChan
		if logger != nil {
			logger.Log("WARN", "Получен сигнал, начинаем graceful shutdown", "signal", sig.String())
		}
		fmt.Printf("\n[INFO] Получен сигнал %s, завершаем работу...\n", sig.String())
		cancel()
	}()

	var operations []*wipe.WipeOperation
	var hasWarnings bool
	var hasErrors bool

	// Обработка дисков с поддержкой отмены
	for _, disk := range targetDisks {
		// Проверка системного диска
		systemPolicy, err := wipe.PrepareSystemDiskWipe(disk.Letter, allowSystemDisk, logger)
		if err != nil {
			logger.Log("ERROR", "Ошибка проверки системного диска", "disk", disk.Letter, "error", err.Error())
			hasErrors = true
			continue
		}

		// Если это системный диск, применяем политику
		if systemPolicy != nil {
			logger.Log("INFO", "Применяется политика системного диска", "disk", disk.Letter, "policy", "system_disk")
		}

		// Проверка контекста перед каждым диском
		select {
		case <-ctx.Done():
			if logger != nil {
				logger.Log("INFO", "Операция отменена пользователем или по таймауту")
			}
			fmt.Println("\n[INFO] Операция отменена")
			break
		default:
		}

		op := wipe.WipeWithEngine(ctx, disk, cfg, logger, dryRun, maxDuration, wipe.WipeEngine(engine), validMode, profile)
		operations = append(operations, op)

		switch op.Status {
		case "COMPLETED":
			// Успешное завершение
		case "PARTIAL":
			hasWarnings = true
			if logger != nil {
				logger.Log("WARN", "Диск обработан частично", "disk", disk.Letter, "reason", op.Warning)
			}
		case "CANCELLED":
			hasWarnings = true
			if logger != nil {
				logger.Log("WARN", "Операция отменена", "disk", disk.Letter, "reason", op.Warning)
			}
		case "FAILED":
			hasErrors = true
			if logger != nil {
				logger.Log("ERROR", "Операция не удалась", "disk", disk.Letter, "error", op.Error)
			}
		}
	}

	// Вывод результатов
	fmt.Println("\nРезультаты затирания:")
	fmt.Println("==================")
	for _, op := range operations {
		status := "✓"
		if op.Status == "PARTIAL" || op.Status == "CANCELLED" {
			status = "⚠"
		} else if op.Status != "COMPLETED" {
			status = "✗"
		}

		fmt.Printf("%s %s - %s (%.1f GB, %.1f MB/s)\n", status, op.Disk, op.Status,
			float64(op.BytesWiped)/(1024*1024*1024), op.SpeedMBps)

		if op.Warning != "" {
			fmt.Printf("  Предупреждение: %s\n", op.Warning)
		}
		if op.Error != "" {
			fmt.Printf("  Ошибка: %s\n", op.Error)
		}
	}

	// Корректные exit codes
	if hasErrors {
		// Генерируем отчёт перед выходом
		endTime := time.Now()
		exitCode := EXIT_ERROR

		if err := generateAndSaveReport(operations, cfg, engine, profile, dryRun, maxDuration, startTime, endTime, exitCode, logger); err != nil {
			logger.Log("WARN", "Ошибка сохранения отчёта", "error", err.Error())
		}

		return fmt.Errorf("некоторые операции завершились с ошибкой")
	}
	if hasWarnings {
		logger.Log("WARN", "Некоторые диски были пропущены")
	}

	// Генерируем отчёт при успешном завершении
	endTime := time.Now()
	if err := generateAndSaveReport(operations, cfg, engine, profile, dryRun, maxDuration, startTime, endTime, EXIT_SUCCESS, logger); err != nil {
		logger.Log("WARN", "Ошибка сохранения отчёта", "error", err.Error())
	}

	return nil
}

func generateAndSaveReport(operations []*wipe.WipeOperation, cfg *config.Config, engine, profile string, dryRun bool, maxDuration time.Duration, startTime, endTime time.Time, exitCode int, logger *logging.EnterpriseLogger) error {
	if cfg != nil && cfg.Reporting.Enabled {
		// Generate legacy report
		report, err := reporting.GenerateReport(operations, cfg, engine, profile, dryRun, maxDuration, startTime, endTime, exitCode)
		if err != nil {
			return fmt.Errorf("ошибка генерации отчёта: %w", err)
		}

		if err := reporting.SaveReport(report, cfg); err != nil {
			return fmt.Errorf("ошибка сохранения отчёта: %w", err)
		}

		logger.Log("INFO", "Отчёт сохранён", "run_id", report.RunID, "file", "wipedisk_report_"+report.Timestamp.Format("20060102_150405")+".json")

		// Generate enterprise security audit report
		enterpriseReport, err := reporting.GenerateEnterpriseReportWrapper(operations, cfg, engine, profile, dryRun, maxDuration, startTime, endTime, exitCode)
		if err != nil {
			logger.Log("WARN", "Ошибка генерации enterprise отчёта", "error", err.Error())
			return nil // Don't fail the operation if enterprise report fails
		}

		// Save enterprise report in both JSON and text formats
		for _, format := range []string{"json", "txt"} {
			if err := reporting.SaveEnterpriseReportWrapper(enterpriseReport, cfg, format); err != nil {
				logger.Log("WARN", "Ошибка сохранения enterprise отчёта", "format", format, "error", err.Error())
			} else {
				logger.Log("INFO", "Enterprise отчёт сохранён", "format", format, "overall_risk", enterpriseReport.Metadata.OverallRisk)
			}
		}
	}
	return nil
}

func runClean(cmd *cobra.Command, args []string) error {
	defaultCfg := config.Default()
	if err := security.SecurityChecks(defaultCfg); err != nil {
		return err
	}

	var err error
	cfg, err = config.Load(configPath)
	if err != nil {
		return err
	}

	var logger *logging.EnterpriseLogger
	logger, err = logging.NewEnterpriseLogger(cfg, verbose)
	if err != nil {
		return err
	}
	defer logger.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()

	return system.CleanTempFiles(ctx, logger, dryRun)
}

func runInfo(cmd *cobra.Command, args []string) error {
	disks, err := system.GetDiskInfo(verbose)
	if err != nil {
		return err
	}

	fmt.Println("Информация о локальных дисках:")
	fmt.Println("==========================")
	for _, disk := range disks {
		fmt.Printf("%s - %s (%.1f GB total, %.1f GB free, %s)\n",
			disk.Letter, disk.Type,
			float64(disk.TotalSize)/(1024*1024*1024),
			float64(disk.FreeSize)/(1024*1024*1024),
			map[bool]string{true: "System", false: "Data"}[disk.IsSystem])
	}

	return nil
}

func runVerify(cmd *cobra.Command, args []string) error {
	// Получаем флаги
	lastSession, _ := cmd.Flags().GetBool("last-session")
	physical, _ := cmd.Flags().GetBool("physical")
	reportPath, _ := cmd.Flags().GetString("report")
	format, _ := cmd.Flags().GetString("format")
	levelStr, _ := cmd.Flags().GetString("level")

	// Валидация уровня проверки
	var level maintenance.VerificationLevel
	switch levelStr {
	case "basic":
		level = maintenance.LevelBasic
	case "physical":
		level = maintenance.LevelPhysical
	case "aggressive":
		level = maintenance.LevelAggressive
	default:
		return fmt.Errorf("неподдерживаемый уровень проверки: %s", levelStr)
	}

	// Если запрошена физическая проверка, повышаем уровень
	if physical && level == maintenance.LevelBasic {
		level = maintenance.LevelPhysical
	}

	// Проверка прав для физической проверки
	if level == maintenance.LevelPhysical || level == maintenance.LevelAggressive {
		if err := security.SecurityChecks(config.Default()); err != nil {
			return fmt.Errorf("физическая проверка требует прав администратора: %w", err)
		}
	}

	// Загружаем конфигурацию
	var err error
	cfg, err = config.Load(configPath)
	if err != nil {
		return fmt.Errorf("ошибка загрузки конфигурации: %w", err)
	}

	// Создаем логгер
	logger, err := logging.NewEnterpriseLogger(cfg, verbose)
	if err != nil {
		return fmt.Errorf("ошибка инициализации логгера: %w", err)
	}
	defer logger.Close()

	logger.Log("INFO", "Запуск верификации", "level", level, "last_session", lastSession)

	// Создаем верификатор
	verifier := maintenance.NewPhysicalVerifier(level, logger)

	// Создаем контекст
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()

	var report *maintenance.VerificationReport

	if lastSession {
		// Проверяем последнюю сессию
		report, err = verifier.VerifyLastSession(ctx)
		if err != nil {
			return fmt.Errorf("ошибка проверки последней сессии: %w", err)
		}
	} else {
		// Ищем последний отчёт автоматически
		report, err = verifier.VerifyLastSession(ctx)
		if err != nil {
			return fmt.Errorf("ошибка поиска отчёта для верификации: %w", err)
		}
	}

	// Выводим результаты
	fmt.Println("\nРезультаты верификации:")
	fmt.Println("======================")
	fmt.Printf("Диск: %s\n", report.Disk)
	fmt.Printf("Метод: %s\n", report.Method)
	fmt.Printf("Проходов: %d\n", report.Passes)
	fmt.Printf("Уровень проверки: %s\n", report.VerificationLevel)
	fmt.Printf("Затирание проверено: %t\n", report.WipeVerified)
	fmt.Printf("Попыток восстановления: %d\n", report.RecoveryAttempts)
	fmt.Printf("Восстановлено данных: %d байт\n", report.RecoveredData)
	fmt.Printf("Успешность: %.1f%%\n", report.SuccessRate)
	fmt.Printf("Длительность: %s\n", report.TestDuration)

	if len(report.Anomalies) > 0 {
		fmt.Println("\nАномалии:")
		for _, anomaly := range report.Anomalies {
			fmt.Printf("  %s (%s): %s [%s]\n", anomaly.Type, anomaly.Severity, anomaly.Description, anomaly.Location)
		}
	}

	if len(report.Compliance) > 0 {
		fmt.Println("\nСоответствие стандартам:")
		for _, standard := range report.Compliance {
			fmt.Printf("  ✓ %s\n", standard)
		}
	}

	// Сохраняем отчёт если нужно
	if reportPath != "" {
		metadata := reporting.VerificationMetadata{
			RunID:       reporting.GenerateRunID(),
			Timestamp:   time.Now(),
			Environment: "production",
			Operator:    os.Getenv("USERNAME"),
			Purpose:     "verification",
		}

		extendedReport := reporting.GenerateVerificationReport(report, metadata)
		if err := reporting.SaveVerificationReport(extendedReport, format, reportPath); err != nil {
			return fmt.Errorf("ошибка сохранения отчёта: %w", err)
		}
		fmt.Printf("\nОтчёт сохранён: %s\n", reportPath)
	}

	// Exit code в зависимости от результатов
	if !report.WipeVerified {
		return fmt.Errorf("верификация не пройдена")
	}

	return nil
}

func runMaintenance(cmd *cobra.Command, args []string) error {
	// Получаем флаги
	planName, _ := cmd.Flags().GetString("plan")
	listPlans, _ := cmd.Flags().GetBool("list-plans")
	silent, _ := cmd.Flags().GetBool("silent")
	parallel, _ := cmd.Flags().GetBool("parallel")

	// Показываем доступные планы
	if listPlans {
		fmt.Println("Доступные планы обслуживания:")
		fmt.Println("=============================")

		plans := maintenance.GetPredefinedPlans()
		for _, plan := range plans {
			fmt.Printf("%-15s - %s\n", plan.Name, plan.Description)
			fmt.Printf("                Фазы: %d, Таймаут: %v, Требует админ: %t\n",
				len(plan.Phases), plan.Timeout, plan.RequireAdmin)
			fmt.Printf("                Параллельно: %t, Тихий режим: %t\n", plan.Parallel, plan.Silent)
			fmt.Println()
		}
		return nil
	}

	if planName == "" {
		return fmt.Errorf("не указан план обслуживания. Используйте --plan для выбора плана или --list-plans для просмотра доступных")
	}

	// Находим план
	plan := maintenance.GetPlanByName(planName)
	if plan == nil {
		return fmt.Errorf("неизвестный план: %s", planName)
	}

	// Проверка прав для планов, требующих админ
	if plan.RequireAdmin {
		if err := security.SecurityChecks(config.Default()); err != nil {
			return fmt.Errorf("план %s требует прав администратора: %w", planName, err)
		}
	}

	// Загружаем конфигурацию
	var err error
	cfg, err = config.Load(configPath)
	if err != nil {
		return fmt.Errorf("ошибка загрузки конфигурации: %w", err)
	}

	// Создаем логгер
	logger, err := logging.NewEnterpriseLogger(cfg, verbose)
	if err != nil {
		return fmt.Errorf("ошибка инициализации логгера: %w", err)
	}
	defer logger.Close()

	// Применяем флаги к плану
	if silent {
		plan.Silent = true
	}
	if parallel {
		plan.Parallel = true
	}

	logger.Log("INFO", "Запуск плана обслуживания",
		"plan", plan.Name,
		"phases", len(plan.Phases),
		"timeout", plan.Timeout,
		"parallel", plan.Parallel,
		"silent", plan.Silent)

	// Создаем оркестратор
	orchestrator := maintenance.NewMaintenanceOrchestrator(cfg, logger, dryRun, verbose)

	// Создаем контекст
	ctx, cancel := context.WithTimeout(context.Background(), plan.Timeout)
	defer cancel()

	// Установка обработчиков сигналов
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		if logger != nil {
			logger.Log("WARN", "Получен сигнал, отмена обслуживания", "signal", sig.String())
		}
		fmt.Printf("\n[INFO] Получен сигнал %s, отмена обслуживания...\n", sig.String())
		cancel()
	}()

	// Выполняем план
	report, err := orchestrator.ExecutePlan(ctx, plan)
	if err != nil {
		return fmt.Errorf("ошибка выполнения плана: %w", err)
	}

	// Выводим результаты
	fmt.Println("\nРезультаты обслуживания:")
	fmt.Println("=======================")
	fmt.Printf("План: %s\n", report.PlanName)
	fmt.Printf("Статус: %s\n", report.Status)
	fmt.Printf("Длительность: %s\n", report.TotalDuration)
	fmt.Printf("Успешных фаз: %d/%d\n", report.SuccessCount, len(report.PhaseResults))
	fmt.Printf("Очищено: %.1f MB\n", float64(report.TotalCleaned)/(1024*1024))

	if len(report.PhaseResults) > 0 {
		fmt.Println("\nДетализация по фазам:")
		for _, result := range report.PhaseResults {
			status := "✓"
			if result.Status == "FAILED" {
				status = "✗"
			} else if result.Status == "SKIPPED" {
				status = "-"
			}

			fmt.Printf("  %s %s - %s (%s", status, result.Phase, result.Status, result.Duration)
			if result.BytesCleaned > 0 {
				fmt.Printf(", %.1f MB", float64(result.BytesCleaned)/(1024*1024))
			}
			fmt.Println(")")

			if result.Error != "" {
				fmt.Printf("    Ошибка: %s\n", result.Error)
			}
		}
	}

	// Exit code в зависимости от результатов
	if report.Status == "FAILED" {
		return fmt.Errorf("обслуживание завершилось с ошибками")
	} else if report.Status == "PARTIAL" {
		logger.Log("WARN", "Обслуживание завершено частично")
	}

	return nil
}

func runDiagnose(cmd *cobra.Command, args []string) error {
	// Получаем флаги
	quick, _ := cmd.Flags().GetBool("quick")
	full, _ := cmd.Flags().GetBool("full")
	deep, _ := cmd.Flags().GetBool("deep")
	testName, _ := cmd.Flags().GetString("test")
	output, _ := cmd.Flags().GetString("output")

	// Определяем уровень диагностики
	var level system.DiagnosticLevel
	switch {
	case quick:
		level = system.LevelQuick
	case full:
		level = system.LevelFull
	case deep:
		level = system.LevelDeep
	default:
		level = system.LevelQuick // По умолчанию
	}

	// Валидация теста
	var test system.DiagnosticTest
	if testName != "" {
		switch testName {
		case "permissions":
			test = system.TestPermissions
		case "disks":
			test = system.TestDisks
		case "memory":
			test = system.TestMemory
		case "cpu":
			test = system.TestCPU
		case "paths":
			test = system.TestPaths
		case "api":
			test = system.TestAPI
		case "wipe":
			test = system.TestWipe
		case "network":
			test = system.TestNetwork
		default:
			return fmt.Errorf("неизвестный тест: %s", testName)
		}
	}

	fmt.Printf("Запуск диагностики системы (уровень: %s)\n", level)

	// Создаем runner
	runner := system.NewSystemDiagnosticsRunner(level, verbose, output, test)

	// Создаем контекст
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Выполняем диагностику
	diagnostics, err := runner.RunDiagnostics(ctx)
	if err != nil {
		return fmt.Errorf("ошибка выполнения диагностики: %w", err)
	}

	// Выводим результаты
	fmt.Println("\nРезультаты диагностики:")
	fmt.Println("=====================")
	fmt.Printf("Уровень: %s\n", diagnostics.Level)
	fmt.Printf("Общий статус: %s\n", diagnostics.Overall)
	fmt.Printf("Длительность: %s\n", diagnostics.Duration)
	fmt.Printf("Всего тестов: %d\n", diagnostics.Summary.TotalTests)
	fmt.Printf("Пройдено: %d\n", diagnostics.Summary.Passed)
	fmt.Printf("Предупреждений: %d\n", diagnostics.Summary.Warnings)
	fmt.Printf("Ошибок: %d\n", diagnostics.Summary.Failed)

	// Информация об окружении
	fmt.Println("\nИнформация об окружении:")
	fmt.Println("------------------------")
	fmt.Printf("ОС: %s\n", diagnostics.Environment.OSVersion)
	fmt.Printf("Архитектура: %s\n", diagnostics.Environment.Architecture)
	fmt.Printf("Пользователь: %s\\%s\n", diagnostics.Environment.Domain, diagnostics.Environment.Username)
	fmt.Printf("Компьютер: %s\n", diagnostics.Environment.MachineName)
	fmt.Printf("Права админа: %t\n", diagnostics.Environment.IsAdmin)
	fmt.Printf("Серверная ОС: %t\n", diagnostics.Environment.IsServer)
	fmt.Printf("CPU ядер: %d\n", diagnostics.Environment.CPUCount)

	// Детальные результаты
	if len(diagnostics.Results) > 0 {
		fmt.Println("\nДетальные результаты:")
		fmt.Println("--------------------")
		for _, result := range diagnostics.Results {
			status := "✓"
			if result.Status == "FAIL" {
				status = "✗"
			} else if result.Status == "WARN" {
				status = "⚠"
			}

			fmt.Printf("%s %s - %s (%v)\n", status, result.Test, result.Message, result.Duration)

			if verbose && result.Details != nil {
				fmt.Printf("   Детали: %+v\n", result.Details)
			}
		}
	}

	// Сохраняем отчёт если нужно
	if output != "" {
		if err := runner.SaveDiagnostics(diagnostics, output); err != nil {
			return fmt.Errorf("ошибка сохранения отчёта: %w", err)
		}
		fmt.Printf("\nОтчёт сохранён: %s\n", output)
	}

	// Exit code в зависимости от результатов
	if diagnostics.Overall == "CRITICAL" {
		return fmt.Errorf("обнаружены критические проблемы")
	} else if diagnostics.Overall == "WARNING" {
		fmt.Println("\n⚠ Обнаружены предупреждения. Рекомендуется проверить систему.")
	}

	return nil
}

func runCleanup(cmd *cobra.Command, args []string) error {
	defaultCfg := config.Default()
	logger, err := logging.NewEnterpriseLogger(defaultCfg, verbose)
	if err != nil {
		return fmt.Errorf("ошибка инициализации логгера: %w", err)
	}
	defer logger.Close()

	cleanupCmd := cli.NewCleanupCommand(logger)

	// Get flags
	listOps, _ := cmd.Flags().GetBool("list")
	category, _ := cmd.Flags().GetString("category")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if listOps {
		return cleanupCmd.ListCleanupOperations()
	}

	if category != "" {
		return cleanupCmd.ExecuteCleanupByCategory(category, dryRun)
	}

	return cleanupCmd.ExecuteCleanup(args, dryRun)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		// Корректные exit codes
		if strings.Contains(err.Error(), "требуются права администратора") ||
			strings.Contains(err.Error(), "запуск на серверных ОС запрещен") ||
			strings.Contains(err.Error(), "ошибка загрузки конфигурации") ||
			strings.Contains(err.Error(), "ошибка создания директорий") {
			os.Exit(EXIT_ERROR)
		} else if strings.Contains(err.Error(), "некоторые операции завершились с ошибкой") {
			os.Exit(EXIT_WARNING)
		} else {
			os.Exit(EXIT_SUCCESS)
		}
	}
	os.Exit(EXIT_SUCCESS)
}
