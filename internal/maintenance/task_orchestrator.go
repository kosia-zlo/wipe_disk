package maintenance

import (
	"context"

	"wipedisk_enterprise/internal/logging"
)

// CreateDefaultTasks создает стандартный набор задач очистки
func CreateDefaultTasks(logger *logging.EnterpriseLogger) []Task {
	return []Task{
		&DNSCleanupTask{},
		NewTempCleanupTask(logger),
		&RecycleBinCleanupTask{},
		&SpoolerCleanupTask{},
	}
}

// CreateCustomTasks создает задачи по списку имен
func CreateCustomTasks(taskNames []string, logger *logging.EnterpriseLogger) []Task {
	var tasks []Task

	for _, name := range taskNames {
		switch name {
		case "dns", "flushdns":
			tasks = append(tasks, &DNSCleanupTask{})
		case "temp", "temporary":
			tasks = append(tasks, NewTempCleanupTask(logger))
		case "recycle", "recyclebin":
			tasks = append(tasks, &RecycleBinCleanupTask{})
		case "spooler", "print":
			tasks = append(tasks, &SpoolerCleanupTask{})
		}
	}

	return tasks
}

// TaskOrchestrator оркестрирует выполнение задач очистки
type TaskOrchestrator struct {
	runner *MaintenanceRunner
	logger *logging.EnterpriseLogger
}

// NewTaskOrchestrator создает новый оркестратор
func NewTaskOrchestrator(logger *logging.EnterpriseLogger) *TaskOrchestrator {
	return &TaskOrchestrator{
		runner: NewMaintenanceRunner(logger),
		logger: logger,
	}
}

// ExecuteDefault выполняет стандартный набор задач
func (to *TaskOrchestrator) ExecuteDefault(ctx context.Context) error {
	tasks := CreateDefaultTasks(to.logger)

	for _, task := range tasks {
		to.runner.AddTask(task)
	}

	return to.runner.Run(ctx)
}

// ExecuteCustom выполняет указанные задачи
func (to *TaskOrchestrator) ExecuteCustom(ctx context.Context, taskNames []string) error {
	tasks := CreateCustomTasks(taskNames, to.logger)

	for _, task := range tasks {
		to.runner.AddTask(task)
	}

	return to.runner.Run(ctx)
}

// ExecuteSingle выполняет одну задачу
func (to *TaskOrchestrator) ExecuteSingle(ctx context.Context, task Task) error {
	to.runner.AddTask(task)
	return to.runner.Run(ctx)
}

// GetAvailableTasks возвращает список доступных задач
func GetAvailableTasks() []string {
	return []string{
		"dns",
		"temp",
		"recycle",
		"spooler",
	}
}
