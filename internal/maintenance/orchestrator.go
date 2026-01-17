package maintenance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"wipedisk_enterprise/internal/config"
	"wipedisk_enterprise/internal/logging"
	"wipedisk_enterprise/internal/system"
	"wipedisk_enterprise/internal/wipe"
)

// MaintenancePhase определяет фазу обслуживания
type MaintenancePhase string

const (
	PhaseCleanTemp     MaintenancePhase = "clean_temp"
	PhaseCleanUpdate   MaintenancePhase = "clean_update_cache"
	PhaseCleanBrowsers MaintenancePhase = "clean_browsers"
	PhaseWipeFreeSpace MaintenancePhase = "wipe_free_space"
	PhaseOptimizeDisk  MaintenancePhase = "optimize_disk"
	PhaseVerifyWipe    MaintenancePhase = "verify_wipe"
)

// MaintenancePlan определяет план обслуживания
type MaintenancePlan struct {
	Name         string             `json:"name"`
	Description  string             `json:"description"`
	Phases       []MaintenancePhase `json:"phases"`
	Timeout      time.Duration      `json:"timeout"`
	Parallel     bool               `json:"parallel"`
	RequireAdmin bool               `json:"require_admin"`
	Silent       bool               `json:"silent"`
}

// PhaseResult содержит результат выполнения фазы
type PhaseResult struct {
	Phase        MaintenancePhase `json:"phase"`
	Status       string           `json:"status"` // COMPLETED, FAILED, SKIPPED
	Duration     time.Duration    `json:"duration"`
	Error        string           `json:"error,omitempty"`
	BytesCleaned uint64           `json:"bytes_cleaned,omitempty"`
	StartTime    time.Time        `json:"start_time"`
	EndTime      time.Time        `json:"end_time"`
}

// MaintenanceReport содержит отчёт о выполнении плана
type MaintenanceReport struct {
	PlanName      string        `json:"plan_name"`
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
	TotalDuration time.Duration `json:"total_duration"`
	Status        string        `json:"status"`
	PhaseResults  []PhaseResult `json:"phase_results"`
	TotalCleaned  uint64        `json:"total_cleaned"`
	SuccessCount  int           `json:"success_count"`
	FailureCount  int           `json:"failure_count"`
}

// MaintenanceOrchestrator управляет выполнением планов обслуживания
type MaintenanceOrchestrator struct {
	logger  *logging.EnterpriseLogger
	config  *config.Config
	dryRun  bool
	verbose bool
}

// NewMaintenanceOrchestrator создает новый оркестратор
func NewMaintenanceOrchestrator(cfg *config.Config, logger *logging.EnterpriseLogger, dryRun, verbose bool) *MaintenanceOrchestrator {
	return &MaintenanceOrchestrator{
		logger:  logger,
		config:  cfg,
		dryRun:  dryRun,
		verbose: verbose,
	}
}

// ExecutePlan выполняет план обслуживания
func (mo *MaintenanceOrchestrator) ExecutePlan(ctx context.Context, plan *MaintenancePlan) (*MaintenanceReport, error) {
	mo.logger.Log("INFO", "Начало выполнения плана обслуживания",
		"plan", plan.Name,
		"phases", len(plan.Phases),
		"timeout", plan.Timeout,
		"parallel", plan.Parallel)

	startTime := time.Now()

	// Создаем контекст с таймаутом для всего плана
	planCtx, cancel := context.WithTimeout(ctx, plan.Timeout)
	defer cancel()

	report := &MaintenanceReport{
		PlanName:     plan.Name,
		StartTime:    startTime,
		PhaseResults: make([]PhaseResult, 0, len(plan.Phases)),
		Status:       "RUNNING",
	}

	var wg sync.WaitGroup
	resultsChan := make(chan PhaseResult, len(plan.Phases))

	if plan.Parallel {
		// Параллельное выполнение
		for _, phase := range plan.Phases {
			wg.Add(1)
			go func(p MaintenancePhase) {
				defer wg.Done()
				result := mo.executePhase(planCtx, p, plan)
				resultsChan <- result
			}(phase)
		}

		// Ожидаем завершения всех горутин
		go func() {
			wg.Wait()
			close(resultsChan)
		}()
	} else {
		// Последовательное выполнение
		go func() {
			for _, phase := range plan.Phases {
				result := mo.executePhase(planCtx, phase, plan)
				resultsChan <- result

				// При последовательном выполнении останавливаемся при ошибке
				if result.Status == "FAILED" && !plan.Silent {
					mo.logger.Log("ERROR", "Фаза завершилась с ошибкой, остановка плана", "phase", phase, "error", result.Error)
					break
				}
			}
			close(resultsChan)
		}()
	}

	// Собираем результаты
	for result := range resultsChan {
		report.PhaseResults = append(report.PhaseResults, result)
		if result.Status == "COMPLETED" {
			report.SuccessCount++
			report.TotalCleaned += result.BytesCleaned
		} else {
			report.FailureCount++
		}
	}

	report.EndTime = time.Now()
	report.TotalDuration = report.EndTime.Sub(report.StartTime)

	// Определяем общий статус
	if report.FailureCount == 0 {
		report.Status = "COMPLETED"
	} else if report.SuccessCount > 0 {
		report.Status = "PARTIAL"
	} else {
		report.Status = "FAILED"
	}

	mo.logger.Log("INFO", "План обслуживания завершен",
		"plan", plan.Name,
		"status", report.Status,
		"duration", report.TotalDuration,
		"success", report.SuccessCount,
		"failed", report.FailureCount,
		"cleaned_mb", report.TotalCleaned/(1024*1024))

	return report, nil
}

// executePhase выполняет отдельную фазу
func (mo *MaintenanceOrchestrator) executePhase(ctx context.Context, phase MaintenancePhase, plan *MaintenancePlan) PhaseResult {
	startTime := time.Now()

	result := PhaseResult{
		Phase:     phase,
		Status:    "RUNNING",
		StartTime: startTime,
	}

	mo.logger.Log("INFO", "Выполнение фазы", "phase", phase, "plan", plan.Name)

	// Создаем контекст для фазы с индивидуальным таймаутом
	phaseTimeout := 30 * time.Minute // По умолчанию
	switch phase {
	case PhaseWipeFreeSpace:
		phaseTimeout = 2 * time.Hour
	case PhaseOptimizeDisk:
		phaseTimeout = 1 * time.Hour
	case PhaseVerifyWipe:
		phaseTimeout = 1 * time.Hour
	}

	phaseCtx, cancel := context.WithTimeout(ctx, phaseTimeout)
	defer cancel()

	var err error
	var bytesCleaned uint64

	// Выполняем фазу
	switch phase {
	case PhaseCleanTemp:
		bytesCleaned, err = mo.cleanTemp(phaseCtx)
	case PhaseCleanUpdate:
		bytesCleaned, err = mo.cleanUpdateCache(phaseCtx)
	case PhaseCleanBrowsers:
		bytesCleaned, err = mo.cleanBrowsers(phaseCtx)
	case PhaseWipeFreeSpace:
		bytesCleaned, err = mo.wipeFreeSpace(phaseCtx, plan)
	case PhaseOptimizeDisk:
		bytesCleaned, err = mo.optimizeDisk(phaseCtx)
	case PhaseVerifyWipe:
		bytesCleaned, err = mo.verifyWipe(phaseCtx)
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.BytesCleaned = bytesCleaned

	if err != nil {
		result.Status = "FAILED"
		result.Error = err.Error()
		mo.logger.Log("ERROR", "Фаза завершилась с ошибкой", "phase", phase, "error", err, "duration", result.Duration)
	} else {
		result.Status = "COMPLETED"
		mo.logger.Log("INFO", "Фаза успешно завершена", "phase", phase, "duration", result.Duration, "cleaned_mb", bytesCleaned/(1024*1024))
	}

	return result
}

// Реализации фаз
func (mo *MaintenanceOrchestrator) cleanTemp(ctx context.Context) (uint64, error) {
	if mo.dryRun {
		mo.logger.Log("INFO", "DRY-RUN: очистка временных файлов")
		return 1024 * 1024 * 1024, nil // 1GB для теста
	}

	err := system.CleanTempFiles(ctx, mo.logger, mo.verbose)
	if err != nil {
		return 0, err
	}

	// В реальной реализации нужно считать размер очищенных файлов
	return 1024 * 1024 * 1024, nil // Заглушка
}

func (mo *MaintenanceOrchestrator) cleanUpdateCache(ctx context.Context) (uint64, error) {
	if mo.dryRun {
		mo.logger.Log("INFO", "DRY-RUN: очистка кэша обновлений")
		return 512 * 1024 * 1024, nil // 512MB для теста
	}

	// Заглушка - реальная реализация будет очищать кэш Windows Update
	return system.CleanUpdateCache(ctx, mo.logger)
}

func (mo *MaintenanceOrchestrator) cleanBrowsers(ctx context.Context) (uint64, error) {
	if mo.dryRun {
		mo.logger.Log("INFO", "DRY-RUN: очистка кэша браузеров")
		return 256 * 1024 * 1024, nil // 256MB для теста
	}

	// Заглушка - реальная реализация будет очищать кэш браузеров
	return system.CleanBrowserCache(ctx, mo.logger)
}

func (mo *MaintenanceOrchestrator) wipeFreeSpace(ctx context.Context, plan *MaintenancePlan) (uint64, error) {
	if mo.dryRun {
		mo.logger.Log("INFO", "DRY-RUN: затирание свободного места")
		return 10 * 1024 * 1024 * 1024, nil // 10GB для теста
	}

	// Получаем диски
	disks, err := system.GetDiskInfo(mo.verbose)
	if err != nil {
		return 0, fmt.Errorf("ошибка получения дисков: %w", err)
	}

	var totalWiped uint64
	for _, disk := range disks {
		// Пропускаем системный диск без явного разрешения
		if disk.IsSystem && !mo.allowSystemDisk() {
			mo.logger.Log("INFO", "Пропуск системного диска", "disk", disk.Letter)
			continue
		}

		// Выполняем затирание
		op := wipe.WipeWithEngine(ctx, disk, mo.config, mo.logger, false, 0, wipe.WipeEngine("internal"), "standard", "balanced")
		if op.Status == "COMPLETED" {
			totalWiped += op.BytesWiped
		} else if op.Error != "" {
			mo.logger.Log("WARN", "Ошибка затирания диска", "disk", disk.Letter, "error", op.Error)
		}
	}

	return totalWiped, nil
}

func (mo *MaintenanceOrchestrator) optimizeDisk(ctx context.Context) (uint64, error) {
	if mo.dryRun {
		mo.logger.Log("INFO", "DRY-RUN: оптимизация диска")
		return 0, nil
	}

	// Заглушка - реальная реализация будет вызывать defrag/TRIM
	return system.OptimizeDisks(ctx, mo.logger)
}

func (mo *MaintenanceOrchestrator) verifyWipe(ctx context.Context) (uint64, error) {
	if mo.dryRun {
		mo.logger.Log("INFO", "DRY-RUN: верификация затирания")
		return 0, nil
	}

	// Используем верификатор из verify.go
	verifier := NewPhysicalVerifier(LevelBasic, mo.logger)
	report, err := verifier.VerifyLastSession(ctx)
	if err != nil {
		return 0, fmt.Errorf("ошибка верификации: %w", err)
	}

	if !report.WipeVerified {
		return 0, fmt.Errorf("верификация не пройдена")
	}

	return 0, nil
}

// allowSystemDisk проверяет разрешено ли затирание системного диска
func (mo *MaintenanceOrchestrator) allowSystemDisk() bool {
	// В реальной реализации нужно проверять конфигурацию
	return false
}

// GetAvailablePlans возвращает доступные планы
func (mo *MaintenanceOrchestrator) GetAvailablePlans() []*MaintenancePlan {
	return GetPredefinedPlans()
}
