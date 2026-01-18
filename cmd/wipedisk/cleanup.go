package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"wipedisk_enterprise/internal/config"
	"wipedisk_enterprise/internal/logging"
	"wipedisk_enterprise/internal/maintenance"
)

// cleanupCmd представляет команду cleanup
var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Очистка системы",
	Long: `Очистка системных временных файлов, DNS кэша, очереди печати и корзины.
Поддерживает выборочные операции и dry-run режим.`,
	Example: `  wipedisk cleanup --all
  wipedisk cleanup --tasks dns,temp
  wipedisk cleanup --list-tasks
  wipedisk cleanup --dry-run`,
	RunE: runCleanupTasks,
}

func init() {
	rootCmd.AddCommand(cleanupCmd)

	cleanupCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Показать что будет сделано без выполнения")
	cleanupCmd.Flags().BoolVar(&verbose, "verbose", false, "Детальный вывод")
	cleanupCmd.Flags().StringSlice("tasks", []string{}, "Список задач для выполнения (dns,temp,recycle,spooler)")
	cleanupCmd.Flags().Bool("all", false, "Выполнить все задачи очистки")
	cleanupCmd.Flags().Bool("list-tasks", false, "Показать список доступных задач")
}

func runCleanupTasks(cmd *cobra.Command, args []string) error {
	// Создаем логгер
	defaultCfg := config.Default()
	logger, err := logging.NewEnterpriseLogger(defaultCfg, verbose)
	if err != nil {
		return fmt.Errorf("ошибка инициализации логгера: %w", err)
	}
	defer logger.Close()

	// Показываем список задач
	listTasks, _ := cmd.Flags().GetBool("list-tasks")
	if listTasks {
		return listAvailableTasks(logger)
	}

	// Получаем параметры
	taskNames, _ := cmd.Flags().GetStringSlice("tasks")
	allTasks, _ := cmd.Flags().GetBool("all")

	if !allTasks && len(taskNames) == 0 {
		return fmt.Errorf("укажите --all или --tasks для выполнения")
	}

	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Создаем оркестратор
	orchestrator := maintenance.NewTaskOrchestrator(logger)

	logger.Log("INFO", "Начало очистки системы", "dry_run", dryRun, "all_tasks", allTasks, "custom_tasks", taskNames)

	if dryRun {
		return performDryRun(ctx, orchestrator, taskNames, allTasks, logger)
	}

	// Выполняем реальные задачи
	if allTasks {
		return orchestrator.ExecuteDefault(ctx)
	}

	return orchestrator.ExecuteCustom(ctx, taskNames)
}

func listAvailableTasks(logger *logging.EnterpriseLogger) error {
	tasks := maintenance.GetAvailableTasks()

	fmt.Println("Доступные задачи очистки:")
	for i, task := range tasks {
		fmt.Printf("  %d. %s\n", i+1, task)
	}

	fmt.Println("\nОписание:")
	fmt.Println("  dns       - Очистка DNS кэша")
	fmt.Println("  temp      - Очистка временных файлов")
	fmt.Println("  recycle   - Очистка корзины")
	fmt.Println("  spooler   - Очистка очереди печати")

	return nil
}

func performDryRun(ctx context.Context, orchestrator *maintenance.TaskOrchestrator, taskNames []string, allTasks bool, logger *logging.EnterpriseLogger) error {
	logger.Log("INFO", "DRY RUN MODE - операции не будут выполнены")

	var tasks []maintenance.Task

	if allTasks {
		tasks = maintenance.CreateDefaultTasks(logger)
	} else {
		tasks = maintenance.CreateCustomTasks(taskNames, logger)
	}

	fmt.Println("\nЗадачи, которые будут выполнены:")
	for i, task := range tasks {
		fmt.Printf("  %d. %s\n", i+1, task.Name())
	}

	fmt.Println("\nДля выполнения без dry-run режима уберите флаг --dry-run")
	return nil
}
