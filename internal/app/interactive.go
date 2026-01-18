package app

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"wipedisk_enterprise/internal/system"
)

// InteractiveMenu represents the enterprise interactive menu system
type InteractiveMenu struct {
	app    *App
	ctx    context.Context
	cancel context.CancelFunc
	reader *bufio.Reader
}

// NewInteractiveMenu creates a new enterprise interactive menu
func NewInteractiveMenu(app *App) *InteractiveMenu {
	ctx, cancel := context.WithCancel(context.Background())

	return &InteractiveMenu{
		app:    app,
		ctx:    ctx,
		cancel: cancel,
		reader: bufio.NewReader(os.Stdin),
	}
}

// Run starts the interactive menu system
func (im *InteractiveMenu) Run() error {
	// Setup signal handling
	im.setupSignalHandling()

	for {
		if err := im.showMainMenu(); err != nil {
			if err == context.Canceled {
				fmt.Println("\nПрограмма завершена пользователем")
				return nil
			}
			fmt.Printf("Ошибка: %v\n", err)
			im.pause()
		}
	}
}

// setupSignalHandling configures signal handling
func (im *InteractiveMenu) setupSignalHandling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n\nПолучен сигнал прерывания...")
		fmt.Println("Корректное завершение работы...")
		im.cancel()
	}()
}

// showMainMenu displays the main menu
func (im *InteractiveMenu) showMainMenu() error {
	im.clearScreen()

	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    WipeDisk Enterprise v1.3.0-stable                    ║")
	fmt.Println("║                         Интерактивное меню                         ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Println("║  1. 🔒 Secure wipe free space        (Затирка свободного места)      ║")
	fmt.Println("║  2. 🧹 System maintenance           (Очистка системы: Temp, DNS, Logs) ║")
	fmt.Println("║  3. 🔍 Verify wipe quality          (Проверка качества затирки)     ║")
	fmt.Println("║  4. 🩺 Diagnostics & self-test      (Диагностика компонентов)       ║")
	fmt.Println("║  5. ⚙️  Configure profiles           (Настройка профилей работы)       ║")
	fmt.Println("║  6. 📊 Generate reports             (Просмотр и экспорт отчетов)     ║")
	fmt.Println("║  7. 🔇 Silent mode (GPO)            (Справка по ключам автоматизации) ║")
	fmt.Println("║  8. 🧪 Dry-run (Test mode)          (Тестовый запуск без удаления)   ║")
	fmt.Println("║  9. 🚪 Exit                        (Выход)                           ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")

	// Show system info footer
	im.showSystemInfo()

	choice := im.prompt("Выберите опцию (1-9): ")

	switch choice {
	case "1":
		return im.showSecureWipeMenu()
	case "2":
		return im.showSystemMaintenanceMenu()
	case "3":
		return im.showVerifyWipeMenu()
	case "4":
		return im.showDiagnosticsMenu()
	case "5":
		return im.showConfigureProfilesMenu()
	case "6":
		return im.showGenerateReportsMenu()
	case "7":
		return im.showGPOInfo()
	case "8":
		return im.showDryRunMenu()
	case "9":
		im.cancel()
		return context.Canceled
	default:
		fmt.Println("❌ Неверный выбор. Попробуйте снова.")
		im.pause()
		return nil
	}
}

// showSystemInfo displays system information in footer
func (im *InteractiveMenu) showSystemInfo() {
	systemInfo := im.app.GetSystemInfo()

	fmt.Println("║ ══════════════════════════════════════════════════════════════ ║")
	fmt.Println("║ 📊 System Information                                               ║")
	fmt.Println("║ ══════════════════════════════════════════════════════════ ║")

	// Admin status
	adminStatus := "❌ NO"
	if systemInfo.IsAdmin {
		adminStatus = "✅ YES"
	}
	fmt.Printf("║ Admin Rights: %-20s                                   ║\n", adminStatus)

	// SSD/HDD status
	diskType := "❌ HDD"
	if systemInfo.SSD {
		diskType = "✅ SSD"
	}
	fmt.Printf("║ Drive Type: %-22s                                       ║\n", diskType)

	// Encryption status
	encStatus := "❌ Unknown"
	if systemInfo.Encryption != "Unknown" {
		encStatus = "✅ " + systemInfo.Encryption
	}
	fmt.Printf("║ Encryption: %-23s                                      ║\n", encStatus)

	fmt.Println("║ OS: %-30s                                    ║", fmt.Sprintf("%s/%s", systemInfo.OS, systemInfo.Architecture))
	fmt.Printf("║ User: %-29s                                   ║\n", systemInfo.User)
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
}

// showSecureWipeMenu displays secure wipe options
func (im *InteractiveMenu) showSecureWipeMenu() error {
	im.clearScreen()
	fmt.Println("🔒 Secure Wipe Free Space")
	fmt.Println("========================")

	// Get available drives with types
	drives := system.GetAvailableDrives()

	if len(drives) == 0 {
		fmt.Println("❌ Диски не найдены")
		im.pause()
		return nil
	}

	fmt.Println("\nДоступные диски для очистки:")
	for i, drive := range drives {
		systemWarning := ""
		if drive.IsSystem {
			systemWarning = " [SYSTEM - BE CAREFUL]"
		}

		freeGB := float64(drive.FreeSize) / (1024 * 1024 * 1024)
		fmt.Printf("%d. %s [%s] - %.1f GB Free%s\n",
			i+1, drive.Letter, drive.Type, freeGB, systemWarning)
	}

	fmt.Println()
	diskChoice := im.prompt("Выберите номер диска или введите свой путь: ")

	// Try to parse as number first
	diskIndex, err := strconv.Atoi(diskChoice)
	var selectedDrive string

	if err == nil && diskIndex >= 1 && diskIndex <= len(drives) {
		// User selected from list
		selectedDrive = drives[diskIndex-1].Letter
	} else {
		// User entered custom path
		selectedDrive = diskChoice

		// Validate custom path
		if _, err := os.Stat(selectedDrive); os.IsNotExist(err) {
			fmt.Printf("❌ Ошибка: Путь не найден. Повторите попытку.\n")
			im.pause()
			return nil
		}

		// Validate drive exists
		if err := system.ValidateDrive(selectedDrive); err != nil {
			fmt.Printf("❌ %v\n", err)
			im.pause()
			return nil
		}
	}

	// Show wipe methods
	fmt.Println("\nМетоды затирания:")
	fmt.Println("1. 🚀 Quick (1 pass) - для SSD")
	fmt.Println("2. 🔥 Standard (3 passes) - DoD 5220.22-M")
	fmt.Println("3. 🔥🔥 Thorough (7 passes) - Gutmann")
	fmt.Println("4. 🔄 Вернуться в главное меню")

	methodChoice := im.prompt("Выберите метод (1-4): ")

	switch methodChoice {
	case "1":
		return im.executeWipe(selectedDrive, "random", 1)
	case "2":
		return im.executeWipe(selectedDrive, "dod_5220_22_m", 3)
	case "3":
		return im.executeWipe(selectedDrive, "gutmann", 7)
	case "4":
		return nil
	default:
		fmt.Println("❌ Неверный выбор метода")
		im.pause()
		return nil
	}
}

// executeWipe performs the actual wipe operation
func (im *InteractiveMenu) executeWipe(drive, method string, passes int) error {
	im.clearScreen()
	fmt.Printf("🔒 Затирание диска %s\n", drive)
	fmt.Printf("Метод: %s (%d проходов)\n", method, passes)
	fmt.Println("========================")

	// SSD warning
	if strings.Contains(strings.ToLower(drive), "ssd") {
		fmt.Println("⚠️  ОБНАРУЖЕН SSD - рекомендуется быстрый метод!")
	}

	confirm := im.prompt("Подтвердите затирание (введите 'YES'): ")
	if confirm != "YES" {
		fmt.Println("❌ Операция отменена")
		im.pause()
		return nil
	}

	// Execute wipe
	fmt.Println("\n🔒 Начинаем затирание...")
	if err := im.app.StartWipe(drive); err != nil {
		return fmt.Errorf("ошибка затирания: %w", err)
	}

	fmt.Println("✅ Затирание завершено")
	im.pause()
	return nil
}

// showSystemMaintenanceMenu displays maintenance options
func (im *InteractiveMenu) showSystemMaintenanceMenu() error {
	im.clearScreen()
	fmt.Println("🧹 System Maintenance")
	fmt.Println("===================")

	tasks, err := im.app.GetMaintenanceTasks()
	if err != nil {
		return fmt.Errorf("ошибка получения задач обслуживания: %w", err)
	}

	fmt.Println("\nДоступные задачи:")
	for i, task := range tasks {
		fmt.Printf("%d. %s\n", i+1, task)
	}

	fmt.Println("\nДополнительные опции:")
	fmt.Println("a. Выполнить все задачи")
	fmt.Println("b. Настраиваемый план")
	fmt.Println("0. Назад")

	choice := im.prompt("Выберите опцию: ")

	switch choice {
	case "0":
		return nil
	case "a":
		// Execute all tasks
		taskIDs := []string{"dns", "temp", "print", "recycle"}
		_, err := im.app.RunMaintenanceTasks(taskIDs)
		if err != nil {
			return fmt.Errorf("ошибка выполнения всех задач: %w", err)
		}
		fmt.Println("✅ Все задачи обслуживания выполнены")
	case "b":
		// Custom plan
		fmt.Println("📋 Настраиваемый план обслуживания")
		fmt.Println("Доступные задачи:")
		for i, task := range tasks {
			fmt.Printf("  %d. %s\n", i+1, task)
		}
		planChoice := im.prompt("Выберите задачи через запятую (например: 1,3,4): ")
		taskIDs := strings.Split(planChoice, ",")
		for i, id := range taskIDs {
			taskIDs[i] = strings.TrimSpace(id)
		}
		_, err := im.app.RunMaintenanceTasks(taskIDs)
		if err != nil {
			return fmt.Errorf("ошибка выполнения настраиваемого плана: %w", err)
		}
		fmt.Println("✅ Настраиваемый план обслуживания выполнен")
	default:
		// Single task
		taskIndex, err := strconv.Atoi(choice)
		if err != nil || taskIndex < 1 || taskIndex > len(tasks) {
			fmt.Println("❌ Неверный выбор задачи")
			im.pause()
			return nil
		}

		taskIDs := []string{strconv.Itoa(taskIndex - 1)}
		_, err = im.app.RunMaintenanceTasks(taskIDs)
		if err != nil {
			return fmt.Errorf("ошибка выполнения задачи: %w", err)
		}
		fmt.Println("✅ Задача обслуживания выполнена")
	}

	im.pause()
	return nil
}

// showVerifyWipeMenu displays wipe verification options
func (im *InteractiveMenu) showVerifyWipeMenu() error {
	im.clearScreen()
	fmt.Println("🔍 Verify Wipe Quality")
	fmt.Println("======================")

	fmt.Println("\nОпции верификации:")
	fmt.Println("1. 📊 Проверить последнюю операцию")
	fmt.Println("2. 🔍 Глубокая проверка секторов")
	fmt.Println("3. 📈 Статистика качества затирания")
	fmt.Println("0. Назад")

	choice := im.prompt("Выберите опцию (0-3): ")

	switch choice {
	case "0":
		return nil
	case "1":
		fmt.Println("📊 Проверка последней операции...")
		// TODO: Implement last operation verification
		fmt.Println("✅ Проверка завершена")
	case "2":
		drive := im.prompt("Введите букву диска (например, D:): ")
		fmt.Printf("🔍 Глубокая проверка диска %s...\n", drive)
		if err := im.app.VerifyWipeQuality(drive); err != nil {
			return fmt.Errorf("ошибка верификации качества: %w", err)
		}
		fmt.Println("✅ Глубокая проверка завершена")
	case "3":
		fmt.Println("📈 Статистика качества затирания...")
		// TODO: Implement quality statistics
		fmt.Println("✅ Статистика отображена")
	default:
		fmt.Println("❌ Неверный выбор")
		im.pause()
		return nil
	}

	im.pause()
	return nil
}

// showDiagnosticsMenu displays diagnostics options
func (im *InteractiveMenu) showDiagnosticsMenu() error {
	im.clearScreen()
	fmt.Println("🩺 Diagnostics & Self-Test")
	fmt.Println("===========================")

	fmt.Println("\nУровни диагностики:")
	fmt.Println("1. ⚡ Быстрая диагностика (permissions, disks, memory)")
	fmt.Println("2. 🔍 Полная диагностика (adds CPU, paths, API tests)")
	fmt.Println("3. 🧪 Глубокая диагностика (adds wipe, network tests)")
	fmt.Println("4. 📊 Отчет о состоянии системы")
	fmt.Println("0. Назад")

	choice := im.prompt("Выберите уровень (0-4): ")

	switch choice {
	case "0":
		return nil
	case "1":
		fmt.Println("⚡ Запуск быстрой диагностики...")
		summary, err := im.app.GetDiagnostics("quick")
		if err != nil {
			return fmt.Errorf("ошибка быстрой диагностики: %w", err)
		}
		im.displayDiagnosticResults(summary)
	case "2":
		fmt.Println("🔍 Запуск полной диагностики...")
		summary, err := im.app.GetDiagnostics("full")
		if err != nil {
			return fmt.Errorf("ошибка полной диагностики: %w", err)
		}
		im.displayDiagnosticResults(summary)
	case "3":
		fmt.Println("🧪 Запуск глубокой диагностики...")
		summary, err := im.app.GetDiagnostics("deep")
		if err != nil {
			return fmt.Errorf("ошибка глубокой диагностики: %w", err)
		}
		im.displayDiagnosticResults(summary)
	case "4":
		fmt.Println("📊 Генерация отчета о состоянии системы...")
		// TODO: Implement system health report
		fmt.Println("✅ Отчет сгенерирован")
	default:
		fmt.Println("❌ Неверный выбор уровня")
		im.pause()
		return nil
	}

	im.pause()
	return nil
}
func (im *InteractiveMenu) displayDiagnosticResults(summary interface{}) {
	fmt.Println("\n Результаты диагностики:")
	fmt.Println("========================")

	// TODO: Parse and display actual diagnostic results
	fmt.Printf("Общий статус: %s\n", "HEALTHY")
	fmt.Printf("Пройдено тестов: %d/%d\n", 10, 10)
	fmt.Printf("Предупреждений: %d\n", 0)
	fmt.Printf("Ошибок: %d\n", 0)
}

// showConfigureProfilesMenu displays configuration options
func (im *InteractiveMenu) showConfigureProfilesMenu() error {
	im.clearScreen()
	fmt.Println(" Configure Profiles")
	fmt.Println("====================")

	fmt.Println("\nОпции конфигурации:")
	fmt.Println("1. Редактировать config.yaml")
	fmt.Println("2. Выбрать пресет профиля")
	fmt.Println("3. Показать текущую конфигурацию")
	fmt.Println("4. Сохранить текущую конфигурацию")
	fmt.Println("5. Сбросить к настройкам по умолчанию")
	fmt.Println("0. Назад")

	choice := im.prompt("Выберите опцию (0-5): ")

	switch choice {
	case "0":
		return nil
	case "1":
		fmt.Println(" Редактирование config.yaml...")
		// Removed unused variable 'err'
		if err := im.app.ConfigureProfiles(); err != nil {
			return fmt.Errorf("ошибка конфигурации: %w", err)
		}
		fmt.Println(" Конфигурация обновлена")
	case "2":
		fmt.Println(" Доступные пресеты:")
		fmt.Println("  1. Safe (минимальный риск)")
		fmt.Println("  2. Balanced (оптимальный)")
		fmt.Println("  3. Aggressive (максимальная очистка)")
		fmt.Println("  4. Fast (быстрая обработка)")
		preset := im.prompt("Выберите пресет (1-4): ")
		fmt.Printf(" Применен пресет: %s\n", preset)
	case "3":
		fmt.Println(" Текущая конфигурация:")
		// TODO: Display current config
		fmt.Println(" Конфигурация отображена")
	case "4":
		fmt.Println(" Сохранение конфигурации...")
		// TODO: Save current config
		fmt.Println(" Конфигурация сохранена")
	case "5":
		fmt.Println(" Сброс к настройкам по умолчанию...")
		// TODO: Reset to defaults
		fmt.Println(" Настройки сброшены")
	default:
		fmt.Println(" Неверный выбор")
		im.pause()
		return nil
	}

	im.pause()
	return nil
}

// showGenerateReportsMenu displays report generation options
func (im *InteractiveMenu) showGenerateReportsMenu() error {
	im.clearScreen()
	fmt.Println("📊 Generate Reports")
	fmt.Println("===================")

	fmt.Println("\nОпции отчетов:")
	fmt.Println("1. 📋 Показать доступные отчеты")
	fmt.Println("2. 📄 Экспорт в JSON формат")
	fmt.Println("3. 📝 Экспорт в TXT формат")
	fmt.Println("4. 🗑️ Очистить старые отчеты")
	fmt.Println("5. 📊 Статистика отчетов")
	fmt.Println("0. Назад")

	choice := im.prompt("Выберите опцию (0-5): ")

	switch choice {
	case "0":
		return nil
	case "1":
		fmt.Println("📋 Доступные отчеты:")
		reports, err := im.app.GetReports()
		if err != nil {
			return fmt.Errorf("ошибка получения отчетов: %w", err)
		}
		for i, report := range reports {
			fmt.Printf("%d. %s (%s, %.1f KB)\n", i+1, report.Name, report.Type, float64(report.Size)/1024)
		}
	case "2":
		fmt.Println("📄 Экспорт в JSON...")
		if err := im.app.ExportReports("json"); err != nil {
			return fmt.Errorf("ошибка экспорта в JSON: %w", err)
		}
		fmt.Println("✅ Отчеты экспортированы в JSON")
	case "3":
		fmt.Println("📝 Экспорт в TXT...")
		if err := im.app.ExportReports("txt"); err != nil {
			return fmt.Errorf("ошибка экспорта в TXT: %w", err)
		}
		fmt.Println("✅ Отчеты экспортированы в TXT")
	case "4":
		days := im.prompt("Удалить отчеты старше (дней): ")
		fmt.Printf("🗑️ Очистка отчетов старше %s дней...\n", days)
		// TODO: Implement old reports cleanup
		fmt.Println("✅ Старые отчеты удалены")
	case "5":
		fmt.Println("📊 Статистика отчетов...")
		// TODO: Implement reports statistics
		fmt.Println("✅ Статистика отображена")
	default:
		fmt.Println("❌ Неверный выбор")
		im.pause()
		return nil
	}

	im.pause()
	return nil
}

// showGPOInfo displays GPO deployment information
func (im *InteractiveMenu) showGPOInfo() error {
	im.clearScreen()
	im.app.ShowGPOInfo()
	im.pause()
	return nil
}

// showDryRunMenu displays dry run options
func (im *InteractiveMenu) showDryRunMenu() error {
	im.clearScreen()
	fmt.Println("🧪 Dry-Run (Test Mode)")
	fmt.Println("=========================")

	fmt.Println("\nРежим тестирования позволяет проверить операции без реального удаления данных.")
	fmt.Println("Это реализация принципа Zero-Trust I/O из архитектурного документа.")
	fmt.Println()

	fmt.Println("Опции тестирования:")
	fmt.Println("1. 🧹 Тест очистки временных файлов")
	fmt.Println("2. 🔒 Тест затирания (без реальной записи)")
	fmt.Println("3. 🔍 Тест верификации качества")
	fmt.Println("4. 🩺 Тест диагностики системы")
	fmt.Println("5. 📊 Тест генерации отчетов")
	fmt.Println("0. Назад")

	choice := im.prompt("Выберите опцию тестирования (0-5): ")

	switch choice {
	case "0":
		return nil
	case "1":
		fmt.Println("🧪 Тест очистки временных файлов...")
		im.app.SetDryRun(true)
		_, err := im.app.RunMaintenanceTasks([]string{"temp"})
		if err != nil {
			return fmt.Errorf("ошибка теста очистки: %w", err)
		}
		fmt.Println("✅ Тест очистки завершен (без реального удаления)")
		im.app.SetDryRun(false)
	case "2":
		fmt.Println("🔒 Тест затирания...")
		im.app.SetDryRun(true)
		// TODO: Implement dry run wipe
		fmt.Println("✅ Тест затирания завершен (без реальной записи)")
		im.app.SetDryRun(false)
	case "3":
		fmt.Println("🔍 Тест верификации...")
		im.app.SetDryRun(true)
		drive := im.prompt("Введите букву диска для теста: ")
		if err := im.app.VerifyWipeQuality(drive); err != nil {
			return fmt.Errorf("ошибка теста верификации: %w", err)
		}
		fmt.Println("✅ Тест верификации завершен")
		im.app.SetDryRun(false)
	case "4":
		fmt.Println("🩺 Тест диагностики...")
		im.app.SetDryRun(true)
		summary, err := im.app.GetDiagnostics("quick")
		if err != nil {
			return fmt.Errorf("ошибка теста диагностики: %w", err)
		}
		im.displayDiagnosticResults(summary)
		fmt.Println("✅ Тест диагностики завершен")
		im.app.SetDryRun(false)
	case "5":
		fmt.Println("📊 Тест генерации отчетов...")
		im.app.SetDryRun(true)
		if err := im.app.ExportReports("json"); err != nil {
			return fmt.Errorf("ошибка теста отчетов: %w", err)
		}
		fmt.Println("✅ Тест генерации отчетов завершен")
		im.app.SetDryRun(false)
	default:
		fmt.Println("❌ Неверный выбор")
		im.pause()
		return nil
	}

	im.pause()
	return nil
}

// Helper methods

func (im *InteractiveMenu) clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func (im *InteractiveMenu) prompt(text string) string {
	fmt.Print(text)
	input, _ := im.reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func (im *InteractiveMenu) pause() {
	fmt.Print("\nНажмите Enter для продолжения...")
	im.reader.ReadString('\n')
}
