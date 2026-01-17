package cli

import (
	"fmt"
	"strings"

	"wipedisk_enterprise/internal/logging"
	"wipedisk_enterprise/internal/system"
)

// CleanupCommand handles cleanup operations
type CleanupCommand struct {
	logger *logging.EnterpriseLogger
}

// NewCleanupCommand creates a new cleanup command
func NewCleanupCommand(logger *logging.EnterpriseLogger) *CleanupCommand {
	return &CleanupCommand{
		logger: logger,
	}
}

// ExecuteCleanup performs cleanup operations
func (c *CleanupCommand) ExecuteCleanup(operations []string, dryRun bool) error {
	if len(operations) == 0 {
		// Default cleanup operations
		operations = []string{
			"Очистка очереди печати",
			"Очистка DNS кэша",
			"Очистка кэша браузеров",
			"Очистка старых логов",
			"Очистка временных файлов",
		}
	}

	c.logger.Log("INFO", "Начало операций очистки", "operations", strings.Join(operations, ", "), "dry_run", dryRun)

	if dryRun {
		c.logger.Log("INFO", "DRY RUN: операции не будут выполнены")
		availableOps := system.GetCleanupOperations()
		for _, op := range operations {
			for _, available := range availableOps {
				if available.Name == op {
					c.logger.Log("INFO", "Будет выполнена операция", "name", op, "category", available.Category, "risk", available.RiskLevel)
					break
				}
			}
		}
		return nil
	}

	results, err := system.ExecuteCleanupOperations(operations)
	if err != nil {
		return fmt.Errorf("ошибка выполнения операций очистки: %w", err)
	}

	// Log results
	successCount := 0
	failureCount := 0

	for _, result := range results {
		if result.Status == "COMPLETED" {
			successCount++
			c.logger.Log("INFO", "Операция очистки выполнена успешно",
				"name", result.Name,
				"duration", result.Duration.String(),
				"category", result.Category)
		} else {
			failureCount++
			c.logger.Log("ERROR", "Операция очистки завершилась с ошибкой",
				"name", result.Name,
				"error", result.Error)
		}
	}

	c.logger.Log("INFO", "Операции очистки завершены",
		"total", len(results),
		"success", successCount,
		"failed", failureCount)

	return nil
}

// ListCleanupOperations lists all available cleanup operations
func (c *CleanupCommand) ListCleanupOperations() error {
	operations := system.GetCleanupOperations()

	fmt.Println("Доступные операции очистки:")
	fmt.Println(strings.Repeat("=", 80))

	for _, op := range operations {
		fmt.Printf("Название: %s\n", op.Name)
		fmt.Printf("Описание: %s\n", op.Description)
		fmt.Printf("Категория: %s\n", op.Category)
		fmt.Printf("Уровень риска: %s\n", op.RiskLevel)
		fmt.Println(strings.Repeat("-", 40))
	}

	return nil
}

// ExecuteCleanupByCategory executes cleanup operations by category
func (c *CleanupCommand) ExecuteCleanupByCategory(category string, dryRun bool) error {
	operations := system.GetCleanupOperations()
	var targetOps []string

	for _, op := range operations {
		if op.Category == category {
			targetOps = append(targetOps, op.Name)
		}
	}

	if len(targetOps) == 0 {
		return fmt.Errorf("категория не найдена: %s", category)
	}

	return c.ExecuteCleanup(targetOps, dryRun)
}

// GetCleanupCategories returns all available cleanup categories
func (c *CleanupCommand) GetCleanupCategories() []string {
	operations := system.GetCleanupOperations()
	categories := make(map[string]bool)

	for _, op := range operations {
		categories[op.Category] = true
	}

	var result []string
	for category := range categories {
		result = append(result, category)
	}

	return result
}
