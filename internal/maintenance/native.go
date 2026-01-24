package maintenance

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"wipedisk_enterprise/internal/logging"
	"wipedisk_enterprise/internal/system"
)

// NativeMaintenance реализует нативные функции обслуживания системы
type NativeMaintenance struct {
	logger *logging.EnterpriseLogger
}

// NewNativeMaintenance создает новый экземпляр NativeMaintenance
func NewNativeMaintenance(logger *logging.EnterpriseLogger) *NativeMaintenance {
	return &NativeMaintenance{logger: logger}
}

// FlushDNS очищает DNS кэш через системный вызов
func (nm *NativeMaintenance) FlushDNS() error {
	nm.logger.Log("INFO", "Начало очистки DNS кэша")

	// Используем ipconfig /flushdns
	cmd := exec.Command("ipconfig", "/flushdns")
	output, err := cmd.CombinedOutput()
	if err != nil {
		nm.logger.Log("ERROR", "Ошибка очистки DNS кэша", "error", err.Error(), "output", string(output))
		return fmt.Errorf("ошибка выполнения ipconfig /flushdns: %w", err)
	}

	nm.logger.Log("INFO", "DNS кэш успешно очищен", "output", strings.TrimSpace(string(output)))
	return nil
}

// CleanTemp безопасно удаляет содержимое временных папок
func (nm *NativeMaintenance) CleanTemp() error {
	nm.logger.Log("INFO", "Начало очистки временных папок")

	// Пути для очистки
	tempPaths := []string{
		os.Getenv("TEMP"),
		os.Getenv("TMP"),
		filepath.Join(system.GetSystemDrive(), "Windows", "Temp"),
	}

	var totalErrors []string
	var totalDeleted int

	for _, tempPath := range tempPaths {
		if tempPath == "" {
			continue
		}

		nm.logger.Log("INFO", "Очистка временной папки", "path", tempPath)

		deleted, errors := nm.cleanDirectory(tempPath)
		totalDeleted += deleted
		totalErrors = append(totalErrors, errors...)
	}

	if len(totalErrors) > 0 {
		nm.logger.Log("WARN", "Очистка завершена с ошибками", "deleted", totalDeleted, "errors", len(totalErrors))
		return fmt.Errorf("ошибки при очистке: %v", totalErrors)
	}

	nm.logger.Log("INFO", "Временные папки успешно очищены", "deleted_files", totalDeleted)
	return nil
}

// ClearPrintSpooler останавливает службу spooler, очищает папку PRINTERS, запускает службу
func (nm *NativeMaintenance) ClearPrintSpooler() error {
	nm.logger.Log("INFO", "Начало очистки очереди печати")

	// Останавливаем службу spooler
	if err := nm.stopService("Spooler"); err != nil {
		nm.logger.Log("ERROR", "Ошибка остановки службы Spooler", "error", err.Error())
		return fmt.Errorf("ошибка остановки службы Spooler: %w", err)
	}

	// Очищаем папку очереди печати
	printerPath := filepath.Join(system.GetSystemDrive(), "Windows", "System32", "spool", "PRINTERS")
	deleted, errors := nm.cleanDirectory(printerPath)

	if len(errors) > 0 {
		nm.logger.Log("WARN", "Ошибки при очистке очереди печати", "errors", errors)
	}

	// Запускаем службу spooler
	if err := nm.startService("Spooler"); err != nil {
		nm.logger.Log("ERROR", "Ошибка запуска службы Spooler", "error", err.Error())
		return fmt.Errorf("ошибка запуска службы Spooler: %w", err)
	}

	nm.logger.Log("INFO", "Очередь печати успешно очищена", "deleted_files", deleted)
	return nil
}

// EmptyRecycleBin очищает корзину на всех дисках через Windows API
func (nm *NativeMaintenance) EmptyRecycleBin() error {
	nm.logger.Log("INFO", "Начало очистки корзины")

	// Используем Windows API через PowerShell
	psScript := `
		Add-Type -AssemblyName Microsoft.VisualBasic
		[Microsoft.VisualBasic.FileIO.FileSystem]::DeleteFile('" + filepath.Join(system.GetSystemDrive(), "$Recycle.Bin", "*") + "','OnlyErrorDialogs','SendToRecycleBin')
	`

	cmd := exec.Command("powershell", "-Command", psScript)
	output, err := cmd.CombinedOutput()
	if err != nil {
		nm.logger.Log("ERROR", "Ошибка очистки корзины", "error", err.Error(), "output", string(output))
		return fmt.Errorf("ошибка очистки корзины: %w", err)
	}

	nm.logger.Log("INFO", "Корзина успешно очищена")
	return nil
}

// cleanDirectory очищает директорию с обработкой ошибок
func (nm *NativeMaintenance) cleanDirectory(dirPath string) (int, []string) {
	var deleted int
	var errors []string

	// Проверяем существование директории
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return 0, []string{fmt.Sprintf("директория не существует: %s", dirPath)}
	}

	// Проходим по всем файлам и поддиректориям
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
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

// stopService останавливает службу Windows
func (nm *NativeMaintenance) stopService(serviceName string) error {
	cmd := exec.Command("sc", "stop", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ошибка остановки службы %s: %w, output: %s", serviceName, err, string(output))
	}

	// Ждем остановки службы
	time.Sleep(2 * time.Second)
	return nil
}

// startService запускает службу Windows
func (nm *NativeMaintenance) startService(serviceName string) error {
	cmd := exec.Command("sc", "start", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ошибка запуска службы %s: %w, output: %s", serviceName, err, string(output))
	}

	// Ждем запуска службы
	time.Sleep(2 * time.Second)
	return nil
}
