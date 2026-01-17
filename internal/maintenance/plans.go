package maintenance

import "time"

// GetPredefinedPlans возвращает предопределенные планы обслуживания
func GetPredefinedPlans() []*MaintenancePlan {
	return []*MaintenancePlan{
		{
			Name:        "full_year",
			Description: "Полное годовое обслуживание (требует админ)",
			Phases: []MaintenancePhase{
				PhaseCleanTemp,
				PhaseCleanUpdate,
				PhaseCleanBrowsers,
				PhaseWipeFreeSpace,
				PhaseOptimizeDisk,
				PhaseVerifyWipe,
			},
			Timeout:      6 * time.Hour,
			Parallel:     false, // Последовательное выполнение для безопасности
			RequireAdmin: true,
			Silent:       false,
		},
		{
			Name:        "light_monthly",
			Description: "Лёгкая ежемесячная очистка",
			Phases: []MaintenancePhase{
				PhaseCleanTemp,
				PhaseCleanBrowsers,
			},
			Timeout:      30 * time.Minute,
			Parallel:     true, // Можно выполнять параллельно
			RequireAdmin: false,
			Silent:       false,
		},
		{
			Name:        "security_quarterly",
			Description: "Квартальное обслуживание безопасности",
			Phases: []MaintenancePhase{
				PhaseCleanTemp,
				PhaseCleanUpdate,
				PhaseWipeFreeSpace,
				PhaseVerifyWipe,
			},
			Timeout:      3 * time.Hour,
			Parallel:     false,
			RequireAdmin: true,
			Silent:       false,
		},
		{
			Name:        "quick_cleanup",
			Description: "Быстрая очистка временных файлов",
			Phases: []MaintenancePhase{
				PhaseCleanTemp,
			},
			Timeout:      15 * time.Minute,
			Parallel:     true,
			RequireAdmin: false,
			Silent:       true,
		},
		{
			Name:        "deep_clean",
			Description: "Глубокая очистка и оптимизация",
			Phases: []MaintenancePhase{
				PhaseCleanTemp,
				PhaseCleanUpdate,
				PhaseCleanBrowsers,
				PhaseWipeFreeSpace,
				PhaseOptimizeDisk,
			},
			Timeout:      4 * time.Hour,
			Parallel:     false,
			RequireAdmin: true,
			Silent:       false,
		},
		{
			Name:        "verify_only",
			Description: "Только верификация последнего затирания",
			Phases: []MaintenancePhase{
				PhaseVerifyWipe,
			},
			Timeout:      1 * time.Hour,
			Parallel:     true,
			RequireAdmin: false,
			Silent:       true,
		},
	}
}

// GetPlanByName находит план по имени
func GetPlanByName(name string) *MaintenancePlan {
	plans := GetPredefinedPlans()
	for _, plan := range plans {
		if plan.Name == name {
			return plan
		}
	}
	return nil
}

// ListPlanNames возвращает список имен доступных планов
func ListPlanNames() []string {
	plans := GetPredefinedPlans()
	names := make([]string, len(plans))
	for i, plan := range plans {
		names[i] = plan.Name
	}
	return names
}
