package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"wipedisk_enterprise/internal/config"
	"wipedisk_enterprise/internal/logging"
	"wipedisk_enterprise/internal/maintenance"
	"wipedisk_enterprise/internal/security"
	"wipedisk_enterprise/internal/system"
	"wipedisk_enterprise/internal/wipe"
)

// InteractiveMenu реализует интерактивное CLI меню
type InteractiveMenu struct {
	ctx    context.Context
	cancel context.CancelFunc
	logger *logging.EnterpriseLogger
}

// NewInteractiveMenu создает новое интерактивное меню
func NewInteractiveMenu() *InteractiveMenu {
	ctx, cancel := context.WithCancel(context.Background())

	// Создаем базовый логгер
	logger, err := logging.NewEnterpriseLogger(config.Default(), false)
	if err != nil {
		fmt.Printf("Ошибка инициализации логгера: %v\n", err)
		os.Exit(1)
	}

	return &InteractiveMenu{
		ctx:    ctx,
		cancel: cancel,
		logger: logger,
	}
}

// Run запускает интерактивное меню
func (im *InteractiveMenu) Run() error {
	// Проверка прав администратора
	if !security.IsAdmin() {
		fmt.Println("ОШИБКА: WipeDisk Enterprise требует прав администратора")
		fmt.Println("Пожалуйста, запустите программу от имени администратора")
		return fmt.Errorf("требуются права администратора")
	}

	// Настройка обработки Ctrl+C
	im.setupSignalHandling()

	for {
		if err := im.showMainMenu(); err != nil {
			if err == context.Canceled {
				fmt.Println("\nПрограмма завершена пользователем")
				return nil
			}
			fmt.Printf("Ошибка: %v\n", err)
		}
	}
}

// setupSignalHandling настраивает обработку сигналов
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

// showMainMenu показывает главное меню
func (im *InteractiveMenu) showMainMenu() error {
	im.clearScreen()
	fmt.Println("==========================================")
	fmt.Println("    WipeDisk Enterprise v1.2.2")
	fmt.Println("    Интерактивное меню")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("1. Затереть диск")
	fmt.Println("2. Системная очистка")
	fmt.Println("3. Диагностика системы")
	fmt.Println("4. Проверка дисков")
	fmt.Println("5. Выход")
	fmt.Println()

	choice := im.prompt("Выберите опцию (1-5): ")

	switch choice {
	case "1":
		return im.showWipeMenu()
	case "2":
		return im.showMaintenanceMenu()
	case "3":
		return im.showDiagnosticsMenu()
	case "4":
		return im.showDiskInfo()
	case "5":
		im.cancel()
		return context.Canceled
	default:
		fmt.Println("Неверный выбор. Попробуйте снова.")
		im.pause()
		return nil
	}
}

// showWipeMenu показывает меню затирания
func (im *InteractiveMenu) showWipeMenu() error {
	im.clearScreen()
	fmt.Println("==========================================")
	fmt.Println("    Затирание диска")
	fmt.Println("==========================================")
	fmt.Println()

	// Получаем список дисков
	disks, err := system.GetDiskInfo(false)
	if err != nil {
		return fmt.Errorf("ошибка получения информации о дисках: %w", err)
	}

	if len(disks) == 0 {
		fmt.Println("Диски не найдены")
		im.pause()
		return nil
	}

	fmt.Println("Доступные диски:")
	for i, disk := range disks {
		status := "Доступен"
		if disk.IsSystem {
			status = "Системный"
		}
		fmt.Printf("%d. %s: %s (%.1f GB свободно) [%s]\n",
			i+1, disk.Letter, disk.Type,
			float64(disk.FreeSize)/(1024*1024*1024), status)
	}
	fmt.Println()

	diskChoice := im.prompt("Выберите диск для затирания (номер): ")
	diskIndex, err := strconv.Atoi(diskChoice)
	if err != nil || diskIndex < 1 || diskIndex > len(disks) {
		fmt.Println("Неверный выбор диска")
		im.pause()
		return nil
	}

	selectedDisk := disks[diskIndex-1]

	// Подтверждение
	fmt.Printf("\nВНИМАНИЕ: Вы выбрали диск %s\n", selectedDisk.Letter)
	fmt.Printf("Свободное место: %.1f GB\n", float64(selectedDisk.FreeSize)/(1024*1024*1024))
	if selectedDisk.IsSystem {
		fmt.Println("⚠️  ЭТО СИСТЕМНЫЙ ДИСК!")
	}

	confirm := im.prompt("Вы уверены? (YES для подтверждения): ")
	if confirm != "YES" {
		fmt.Println("Операция отменена")
		im.pause()
		return nil
	}

	// Выполнение затирания
	return im.performWipe(selectedDisk)
}

// showMaintenanceMenu показывает меню обслуживания
func (im *InteractiveMenu) showMaintenanceMenu() error {
	im.clearScreen()
	fmt.Println("==========================================")
	fmt.Println("    Системная очистка")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("1. Очистить DNS кэш")
	fmt.Println("2. Очистить временные файлы")
	fmt.Println("3. Очистить очередь печати")
	fmt.Println("4. Очистить корзину")
	fmt.Println("5. Выполнить все операции")
	fmt.Println("6. Назад")
	fmt.Println()

	choice := im.prompt("Выберите опцию (1-6): ")

	native := maintenance.NewNativeMaintenance(im.logger)

	switch choice {
	case "1":
		return im.performMaintenance("Очистка DNS кэша", native.FlushDNS)
	case "2":
		return im.performMaintenance("Очистка временных файлов", native.CleanTemp)
	case "3":
		return im.performMaintenance("Очистка очереди печати", native.ClearPrintSpooler)
	case "4":
		return im.performMaintenance("Очистка корзины", native.EmptyRecycleBin)
	case "5":
		return im.performAllMaintenance(native)
	case "6":
		return nil
	default:
		fmt.Println("Неверный выбор. Попробуйте снова.")
		im.pause()
		return nil
	}
}

// showDiagnosticsMenu показывает меню диагностики
func (im *InteractiveMenu) showDiagnosticsMenu() error {
	im.clearScreen()
	fmt.Println("==========================================")
	fmt.Println("    Диагностика системы")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("1. Быстрая диагностика")
	fmt.Println("2. Полная диагностика")
	fmt.Println("3. Проверка безопасности")
	fmt.Println("4. Назад")
	fmt.Println()

	choice := im.prompt("Выберите опцию (1-4): ")

	switch choice {
	case "1":
		return im.runDiagnostics("quick")
	case "2":
		return im.runDiagnostics("full")
	case "3":
		return im.runDiagnostics("security")
	case "4":
		return nil
	default:
		fmt.Println("Неверный выбор. Попробуйте снова.")
		im.pause()
		return nil
	}
}

// showDiskInfo показывает информацию о дисках
func (im *InteractiveMenu) showDiskInfo() error {
	im.clearScreen()
	fmt.Println("==========================================")
	fmt.Println("    Информация о дисках")
	fmt.Println("==========================================")
	fmt.Println()

	disks, err := system.GetDiskInfo(false)
	if err != nil {
		return fmt.Errorf("ошибка получения информации о дисках: %w", err)
	}

	for _, disk := range disks {
		fmt.Printf("Диск: %s\n", disk.Letter)
		fmt.Printf("  Тип: %s\n", disk.Type)
		fmt.Printf("  Всего: %.1f GB\n", float64(disk.TotalSize)/(1024*1024*1024))
		fmt.Printf("  Свободно: %.1f GB\n", float64(disk.FreeSize)/(1024*1024*1024))
		fmt.Printf("  Системный: %t\n", disk.IsSystem)
		fmt.Printf("  Доступен для записи: %t\n", disk.IsWritable)
		fmt.Println()
	}

	im.pause()
	return nil
}

// performWipe выполняет затирание диска
func (im *InteractiveMenu) performWipe(disk system.DiskInfo) error {
	fmt.Printf("\nНачинаем затирание диска %s...\n", disk.Letter)

	// Создаем конфигурацию для Persistent File Engine
	progressChan := make(chan wipe.ProgressInfo, 100)

	config := &wipe.PersistentFileConfig{
		BufferSize:  1024 * 1024, // 1MB
		MaxDuration: 0,           // Без ограничений
		Progress:    progressChan,
		Logger:      im.logger,
		Pattern:     nil, // Случайные данные
	}

	// Создаем движок
	engine := wipe.NewPersistentFileEngine(config)

	// Запускаем прогресс в отдельной горутине
	go im.showProgress(progressChan)

	// Выполняем затирание
	result, err := engine.Wipe(im.ctx, disk.Letter, nil)
	if err != nil {
		return fmt.Errorf("ошибка затирания: %w", err)
	}

	fmt.Printf("\n✅ Затирание завершено!\n")
	fmt.Printf("Записано: %.1f GB\n", float64(result.BytesWritten)/(1024*1024*1024))
	fmt.Printf("Скорость: %.1f MB/s\n", result.SpeedMBps)
	fmt.Printf("Длительность: %v\n", result.Duration)

	im.pause()
	return nil
}

// performMaintenance выполняет одну операцию обслуживания
func (im *InteractiveMenu) performMaintenance(name string, operation func() error) error {
	fmt.Printf("\nВыполняется: %s\n", name)

	start := time.Now()
	err := operation()
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("❌ Ошибка: %v\n", err)
	} else {
		fmt.Printf("✅ Завершено за %v\n", duration)
	}

	im.pause()
	return nil
}

// performAllMaintenance выполняет все операции обслуживания
func (im *InteractiveMenu) performAllMaintenance(native *maintenance.NativeMaintenance) error {
	fmt.Println("\nВыполняется полная системная очистка...")

	operations := []struct {
		name string
		fn   func() error
	}{
		{"Очистка DNS кэша", native.FlushDNS},
		{"Очистка временных файлов", native.CleanTemp},
		{"Очистка очереди печати", native.ClearPrintSpooler},
		{"Очистка корзины", native.EmptyRecycleBin},
	}

	for _, op := range operations {
		fmt.Printf("\n• %s...", op.name)
		if err := op.fn(); err != nil {
			fmt.Printf(" ❌\n")
		} else {
			fmt.Printf(" ✅\n")
		}
	}

	fmt.Println("\n✅ Полная очистка завершена")
	im.pause()
	return nil
}

// runDiagnostics запускает диагностику
func (im *InteractiveMenu) runDiagnostics(mode string) error {
	fmt.Printf("\nЗапускается диагностика: %s\n", mode)

	// Здесь будет интеграция с существующим модулем диагностики
	fmt.Println("🔍 Диагностика в разработке...")

	im.pause()
	return nil
}

// showProgress показывает прогресс затирания
func (im *InteractiveMenu) showProgress(progressChan <-chan wipe.ProgressInfo) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case progress := <-progressChan:
			fmt.Printf("\rЗаписано: %.1f GB | Скорость: %.1f MB/s | Прогресс: %.1f%%",
				float64(progress.BytesWritten)/(1024*1024*1024),
				progress.SpeedMBps,
				progress.Percentage)
		case <-ticker.C:
			// Обновляем каждую секунду
		case <-im.ctx.Done():
			return
		}
	}
}

// Вспомогательные функции
func (im *InteractiveMenu) clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func (im *InteractiveMenu) prompt(message string) string {
	fmt.Print(message)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func (im *InteractiveMenu) pause() {
	fmt.Print("\nНажмите Enter для продолжения...")
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')
}

// checkInteractiveMode проверяет, нужно ли запускать интерактивное меню
func checkInteractiveMode() bool {
	// Если нет аргументов командной строки - запускаем интерактивное меню
	return len(os.Args) == 1
}

// initInteractiveMode инициализирует и запускает интерактивное меню
func initInteractiveMode() {
	menu := NewInteractiveMenu()
	if err := menu.Run(); err != nil {
		fmt.Printf("Ошибка интерактивного меню: %v\n", err)
		os.Exit(1)
	}
}
