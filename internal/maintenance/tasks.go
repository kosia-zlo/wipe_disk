package maintenance

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"

	"wipedisk_enterprise/internal/logging"
)

// Task представляет задачу по очистке системы
type Task interface {
	Name() string
	Execute(ctx context.Context) error
}

// MaintenanceRunner выполняет список задач очистки
type MaintenanceRunner struct {
	tasks  []Task
	logger *logging.EnterpriseLogger
}

// NewMaintenanceRunner создает новый экземпляр MaintenanceRunner
func NewMaintenanceRunner(logger *logging.EnterpriseLogger) *MaintenanceRunner {
	return &MaintenanceRunner{
		tasks:  make([]Task, 0),
		logger: logger,
	}
}

// AddTask добавляет задачу в список выполнения
func (mr *MaintenanceRunner) AddTask(task Task) {
	mr.tasks = append(mr.tasks, task)
}

// Run выполняет все задачи последовательно
func (mr *MaintenanceRunner) Run(ctx context.Context) error {
	mr.logger.Log("INFO", "Начало выполнения задач обслуживания", "tasks_count", len(mr.tasks))

	for i, task := range mr.tasks {
		select {
		case <-ctx.Done():
			mr.logger.Log("WARN", "Выполнение прервано", "task_index", i, "task_name", task.Name())
			return ctx.Err()
		default:
		}

		mr.logger.Log("INFO", "Выполнение задачи", "task_index", i+1, "task_name", task.Name())

		start := time.Now()
		err := task.Execute(ctx)
		duration := time.Since(start)

		if err != nil {
			mr.logger.Log("ERROR", "Задача завершилась с ошибкой",
				"task_index", i+1,
				"task_name", task.Name(),
				"error", err.Error(),
				"duration", duration)
		} else {
			mr.logger.Log("INFO", "Задача успешно выполнена",
				"task_index", i+1,
				"task_name", task.Name(),
				"duration", duration)
		}
	}

	mr.logger.Log("INFO", "Все задачи обслуживания завершены")
	return nil
}

// DNSCleanupTask очищает DNS кэш
type DNSCleanupTask struct{}

// Name возвращает имя задачи
func (t *DNSCleanupTask) Name() string {
	return "DNS Cache Cleanup"
}

// Execute выполняет очистку DNS кэша
func (t *DNSCleanupTask) Execute(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Используем ipconfig /flushdns
	cmd := exec.CommandContext(ctx, "ipconfig", "/flushdns")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ошибка выполнения ipconfig /flushdns: %w, output: %s", err, string(output))
	}

	// Проверяем успешность выполнения
	outputStr := strings.ToLower(string(output))
	if strings.Contains(outputStr, "successfully flushed") ||
		strings.Contains(outputStr, "успешно очищен") ||
		strings.Contains(outputStr, "flushed successfully") ||
		strings.Contains(outputStr, "dns cache") ||
		err == nil {
		return nil
	}

	// Если есть ошибка, но команда выполнилась, считаем успехом
	if strings.Contains(outputStr, "dns") && !strings.Contains(outputStr, "error") {
		return nil
	}

	return fmt.Errorf("очистка DNS кэша может не удалась: %s", string(output))
}

// TempCleanupTask очищает временные файлы
type TempCleanupTask struct {
	logger *logging.EnterpriseLogger
}

// Name возвращает имя задачи
func (t *TempCleanupTask) Name() string {
	return "Temporary Files Cleanup"
}

// NewTempCleanupTask создает новую задачу очистки временных файлов
func NewTempCleanupTask(logger *logging.EnterpriseLogger) *TempCleanupTask {
	return &TempCleanupTask{logger: logger}
}

// Execute выполняет очистку временных файлов
func (t *TempCleanupTask) Execute(ctx context.Context) error {
	tempPaths := []string{
		os.TempDir(),
		`C:\Windows\Temp`,
		// Windows 10/11 specific paths
		`C:\Windows\SoftwareDistribution\Download`,                   // Кэш обновлений Windows
		os.Getenv("LOCALAPPDATA") + `\Microsoft\Windows\Explorer`,    // Кэш эскизов и иконок
		os.Getenv("LOCALAPPDATA") + `\Microsoft\Windows\INetCache`,   // Кэш Internet Explorer
		os.Getenv("LOCALAPPDATA") + `\Microsoft\Windows\INetCookies`, // Cookies Internet Explorer
		os.Getenv("LOCALAPPDATA") + `\Temp`,                          // Локальный temp
		os.Getenv("TEMP"),                                            // Системный temp
		os.Getenv("TMP"),                                             // Альтернативный temp
	}

	var totalErrors []string
	var totalDeleted int

	for _, tempPath := range tempPaths {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Проверяем существование пути
		if _, err := os.Stat(tempPath); os.IsNotExist(err) {
			continue // Пропускаем несуществующие пути
		}

		deleted, errors := t.cleanDirectory(ctx, tempPath)
		totalDeleted += deleted
		totalErrors = append(totalErrors, errors...)
	}

	if len(totalErrors) > 0 {
		t.logger.Log("WARN", "Очистка завершена с ошибками", "deleted", totalDeleted, "errors", len(totalErrors))
		return fmt.Errorf("ошибки при очистке: %v", totalErrors)
	}

	t.logger.Log("INFO", "Временные файлы успешно очищены", "deleted_files", totalDeleted)
	return nil
}

// cleanDirectory очищает директорию с обработкой ошибок
func (t *TempCleanupTask) cleanDirectory(ctx context.Context, dirPath string) (int, []string) {
	var deleted int
	var errors []string

	// Проверяем существование директории
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return 0, []string{fmt.Sprintf("директория не существует: %s", dirPath)}
	}

	// Проходим по всем файлам и поддиректориям
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			errors = append(errors, fmt.Sprintf("ошибка доступа к %s: %v", path, err))
			return nil // Продолжаем обход
		}

		// Пропускаем корневую директорию
		if path == dirPath {
			return nil
		}

		// Пробуем удалить файл/директорию
		var removeErr error
		if info.IsDir() {
			// Небольшая задержка для освобождения файлов
			time.Sleep(10 * time.Millisecond)
			removeErr = os.RemoveAll(path)
		} else {
			removeErr = os.Remove(path)
		}

		if removeErr != nil {
			// Игнорируем ошибки для файлов, которые заняты
			if !isFileInUseError(removeErr) {
				errors = append(errors, fmt.Sprintf("ошибка удаления %s: %v", path, removeErr))
			}
		} else {
			deleted++
		}

		return nil
	})

	if err != nil {
		errors = append(errors, fmt.Sprintf("ошибка обхода директории %s: %v", dirPath, err))
	}

	return deleted, errors
}

// RecycleBinCleanupTask очищает корзину
type RecycleBinCleanupTask struct{}

// Name возвращает имя задачи
func (t *RecycleBinCleanupTask) Name() string {
	return "Recycle Bin Cleanup"
}

// Execute выполняет очистку корзины через Shell32.dll
func (t *RecycleBinCleanupTask) Execute(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Загружаем Shell32.dll
	shell32, err := windows.LoadDLL("shell32.dll")
	if err != nil {
		return fmt.Errorf("ошибка загрузки shell32.dll: %w", err)
	}

	// Получаем процедуру SHEmptyRecycleBinW
	emptyRecycleBin, err := shell32.FindProc("SHEmptyRecycleBinW")
	if err != nil {
		return fmt.Errorf("ошибка поиска процедуры SHEmptyRecycleBinW: %w", err)
	}

	// Вызываем SHEmptyRecycleBinW
	// Параметры: hwnd, pszRootPath, dwFlags
	// Используем SHERB_NOCONFIRMATION | SHERB_NOPROGRESSUI | SHERB_NOSOUND
	const (
		SHERB_NOCONFIRMATION = 0x00000001
		SHERB_NOPROGRESSUI   = 0x00000002
		SHERB_NOSOUND        = 0x00000004
	)

	ret, _, err := emptyRecycleBin.Call(
		0, // hwnd (null)
		0, // pszRootPath (null - все диски)
		uintptr(SHERB_NOCONFIRMATION|SHERB_NOPROGRESSUI|SHERB_NOSOUND),
	)

	if ret != 0 {
		// Возвращаемое значение 0 означает успех
		return fmt.Errorf("ошибка очистки корзины: %v", err)
	}

	return nil
}

// SpoolerCleanupTask очищает очередь печати
type SpoolerCleanupTask struct{}

// Name возвращает имя задачи
func (t *SpoolerCleanupTask) Name() string {
	return "Print Spooler Cleanup"
}

// Execute выполняет очистку очереди печати
func (t *SpoolerCleanupTask) Execute(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Останавливаем службу spooler
	if err := t.stopService(ctx, "Spooler"); err != nil {
		return fmt.Errorf("ошибка остановки службы Spooler: %w", err)
	}

	// Очищаем папку очереди печати
	printerPath := `C:\Windows\System32\spool\PRINTERS`
	_, errors := t.cleanSpoolerDirectory(ctx, printerPath)

	if len(errors) > 0 {
		return fmt.Errorf("ошибки при очистке очереди печати: %v", errors)
	}

	// Запускаем службу spooler
	if err := t.startService(ctx, "Spooler"); err != nil {
		return fmt.Errorf("ошибка запуска службы Spooler: %w", err)
	}

	return nil
}

// stopService останавливает службу Windows
func (t *SpoolerCleanupTask) stopService(ctx context.Context, serviceName string) error {
	cmd := exec.CommandContext(ctx, "sc", "stop", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ошибка остановки службы %s: %w, output: %s", serviceName, err, string(output))
	}

	// Ждем остановки службы
	time.Sleep(2 * time.Second)
	return nil
}

// startService запускает службу Windows
func (t *SpoolerCleanupTask) startService(ctx context.Context, serviceName string) error {
	cmd := exec.CommandContext(ctx, "sc", "start", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ошибка запуска службы %s: %w, output: %s", serviceName, err, string(output))
	}

	// Ждем запуска службы
	time.Sleep(2 * time.Second)
	return nil
}

// cleanSpoolerDirectory очищает директорию спулера
func (t *SpoolerCleanupTask) cleanSpoolerDirectory(ctx context.Context, dirPath string) (int, []string) {
	var deleted int
	var errors []string

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			errors = append(errors, fmt.Sprintf("ошибка доступа к %s: %v", path, err))
			return nil
		}

		// Пропускаем корневую директорию
		if path == dirPath {
			return nil
		}

		// Удаляем файлы и директории
		removeErr := os.RemoveAll(path)
		if removeErr != nil {
			errors = append(errors, fmt.Sprintf("ошибка удаления %s: %v", path, removeErr))
		} else {
			deleted++
		}

		return nil
	})

	if err != nil {
		errors = append(errors, fmt.Sprintf("ошибка обхода директории %s: %v", dirPath, err))
	}

	return deleted, errors
}

// Вспомогательные функции

// isFileInUseError проверяет, является ли ошибка ошибкой занятости файла
func isFileInUseError(err error) bool {
	if err == nil {
		return false
	}

	// Проверяем Windows ошибки
	if pathErr, ok := err.(*os.PathError); ok {
		if errno, ok := pathErr.Err.(syscall.Errno); ok {
			// ERROR_SHARING_VIOLATION (32) - файл используется другим процессом
			// ERROR_LOCK_VIOLATION (33) - файл заблокирован
			return errno == 32 || errno == 33
		}
	}

	return strings.Contains(err.Error(), "being used by another process") ||
		strings.Contains(err.Error(), "access is denied")
}
