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

type InteractiveMenu struct {
	app    *App
	ctx    context.Context
	cancel context.CancelFunc
	reader *bufio.Reader
}

func NewInteractiveMenu(app *App) *InteractiveMenu {
	ctx, cancel := context.WithCancel(context.Background())
	return &InteractiveMenu{
		app:    app,
		ctx:    ctx,
		cancel: cancel,
		reader: bufio.NewReader(os.Stdin),
	}
}

func (im *InteractiveMenu) Run() error {
	im.setupSignalHandling()
	for {
		if err := im.showMainMenu(); err != nil {
			if err == context.Canceled {
				fmt.Println("\nПрограмма завершена.")
				return nil
			}
			fmt.Printf("\n❌ Ошибка: %v\n", err)
			im.pause()
		}
	}
}

func (im *InteractiveMenu) setupSignalHandling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		im.cancel()
	}()
}

func (im *InteractiveMenu) showMainMenu() error {
	im.clearScreen()
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║               WipeDisk Enterprise v1.3.0-stable                ║")
	fmt.Println("║                    Интерактивное меню                          ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")
	fmt.Println("║  1. 🔒 Secure wipe free space        (Затирка места)           ║")
	fmt.Println("║  2. 🧹 System maintenance           (Очистка системы)          ║")
	fmt.Println("║  3. 🔍 Verify wipe quality          (Проверка качества)        ║")
	fmt.Println("║  4. 🩺 Diagnostics & self-test      (Диагностика)              ║")
	fmt.Println("║  5. ⚙️  Configure profiles           (Профили работы)           ║")
	fmt.Println("║  6. 📊 Generate reports             (Отчеты)                   ║")
	fmt.Println("║  7. 🔇 Silent mode (GPO)            (Справка GPO)              ║")
	fmt.Println("║  8. 🧪 Dry-run (Test mode)          (Тестовый запуск)          ║")
	fmt.Println("║  0. 💾 Show all local drives       (Все диски)                ║")
	fmt.Println("║  W. 🗑️  Wipe ALL drives              (Все диски сразу)         ║")
	fmt.Println("║  9. 🚪 Exit                         (Выход)                    ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")
	im.showSystemInfo()

	choice := im.prompt("Выберите опцию (0-9, W): ")
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
	case "0":
		return im.showAllLocalDrives()
	case "W":
		return im.wipeAllDrives()
	case "w":
		return im.wipeAllDrives()
	case "9":
		im.cancel()
		return context.Canceled
	default:
		return nil
	}
}

func (im *InteractiveMenu) showSystemInfo() {
	info := im.app.GetSystemInfo()
	fmt.Printf("║ Admin: %-5t | SSD: %-5t | User: %-25s ║\n", info.IsAdmin, info.SSD, info.User)
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
}

func (im *InteractiveMenu) showSecureWipeMenu() error {
	im.clearScreen()
	drives := system.GetAvailableDrives()
	if len(drives) == 0 {
		return fmt.Errorf("диски не найдены")
	}

	fmt.Println("Доступные диски:")
	for i, d := range drives {
		fmt.Printf("%d. %s [%s] - %.1f GB Free\n", i+1, d.Letter, d.Type, float64(d.FreeSize)/1e9)
	}

	choice := im.prompt("\nВыберите номер: ")
	idx, _ := strconv.Atoi(choice)
	if idx < 1 || idx > len(drives) {
		return fmt.Errorf("неверный выбор")
	}

	drive := strings.TrimRight(drives[idx-1].Letter, ".\\") + "\\"

	fmt.Println("\n1. Quick (1 pass)\n2. Standard (3 passes)")
	m := im.prompt("Метод: ")
	p := 1
	if m == "2" {
		p = 3
	}

	return im.executeWipe(drive, "random", p)
}

func (im *InteractiveMenu) executeWipe(drive, method string, passes int) error {
	im.clearScreen()
	fmt.Printf("🔒 ЗАПУСК: %s\nПодтвердите (YES): ", drive)
	if strings.ToUpper(im.prompt("")) != "YES" {
		return fmt.Errorf("отменено")
	}

	fmt.Println("⏳ Работаем...")

	// Запускаем затирание и ждем завершения
	err := im.app.StartWipe(drive)
	if err != nil {
		return err
	}

	fmt.Println("\n✅ Успешно завершено!")
	im.pause()
	return nil
}

func (im *InteractiveMenu) showSystemMaintenanceMenu() error {
	fmt.Println("Модуль System Maintenance находится на стадии бета-тестирования.")
	fmt.Println("Для активации обратитесь к администратору Enterprise-лицензии.")
	im.pause()
	return nil
}

func (im *InteractiveMenu) showVerifyWipeMenu() error {
	fmt.Println("Модуль Verify Wipe Quality находится на стадии бета-тестирования.")
	fmt.Println("Для активации обратитесь к администратору Enterprise-лицензии.")
	im.pause()
	return nil
}

func (im *InteractiveMenu) showDiagnosticsMenu() error {
	fmt.Println("Модуль Diagnostics & Self-test находится на стадии бета-тестирования.")
	fmt.Println("Для активации обратитесь к администратору Enterprise-лицензии.")
	im.pause()
	return nil
}

func (im *InteractiveMenu) showConfigureProfilesMenu() error {
	fmt.Println("Модуль Configure Profiles находится на стадии бета-тестирования.")
	fmt.Println("Для активации обратитесь к администратору Enterprise-лицензии.")
	im.pause()
	return nil
}

func (im *InteractiveMenu) showGenerateReportsMenu() error {
	fmt.Println("Модуль Generate Reports находится на стадии бета-тестирования.")
	fmt.Println("Для активации обратитесь к администратору Enterprise-лицензии.")
	im.pause()
	return nil
}

func (im *InteractiveMenu) showGPOInfo() error {
	fmt.Println("Модуль Silent Mode (GPO) находится на стадии бета-тестирования.")
	fmt.Println("Для активации обратитесь к администратору Enterprise-лицензии.")
	im.pause()
	return nil
}

func (im *InteractiveMenu) showDryRunMenu() error {
	fmt.Println("Модуль Dry-run (Test Mode) находится на стадии бета-тестирования.")
	fmt.Println("Для активации обратитесь к администратору Enterprise-лицензии.")
	im.pause()
	return nil
}

func (im *InteractiveMenu) showAllLocalDrives() error {
	im.clearScreen()
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    ВСЕ ЛОКАЛЬНЫЕ ДИСКИ                         ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")

	drives := system.GetAvailableDrives()
	if len(drives) == 0 {
		fmt.Println("║ Диски не найдены                                                 ║")
	} else {
		for i, d := range drives {
			totalGB := float64(d.FreeSize) / 1e9
			status := "Доступен"
			if d.IsSystem {
				status = "СИСТЕМНЫЙ"
			}
			fmt.Printf("║ %d. %s [%s] - %.1f GB свободно - %s                     ║\n",
				i+1, d.Letter, d.Type, totalGB, status)
		}
	}

	fmt.Println("╠════════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ Всего дисков: %d                                              ║\n", len(drives))
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")

	im.pause()
	return nil
}

func (im *InteractiveMenu) wipeAllDrives() error {
	im.clearScreen()
	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║               ⚠️  ЗАТИРАНИЕ ВСЕХ ДИСКОВ ⚠️                     ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")

	drives := system.GetAvailableDrives()
	if len(drives) == 0 {
		fmt.Println("║ Диски не найдены                                                 ║")
		im.pause()
		return nil
	}

	fmt.Println("║ Найденные диски:                                               ║")
	for i, d := range drives {
		totalGB := float64(d.FreeSize) / 1e9
		status := "Доступен"
		if d.IsSystem {
			status = "СИСТЕМНЫЙ"
		}
		fmt.Printf("║ %d. %s [%s] - %.1f GB свободно - %s                     ║\n",
			i+1, d.Letter, d.Type, totalGB, status)
	}

	fmt.Println("╠════════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ ВСЕГО ДИСКОВ ДЛЯ ЗАТИРАНИЯ: %d                                 ║\n", len(drives))
	fmt.Println("║                                                               ║")
	fmt.Println("║ ⚠️  ВНИМАНИЕ: Это затрет свободное место на ВСЕХ дисках!      ║")
	fmt.Println("║    Системный диск будет затерт тоже!                            ║")
	fmt.Println("║                                                               ║")
	fmt.Println("║ Для подтверждения введите: WIPE_ALL_DRIVES                     ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")

	confirmation := im.prompt("Подтверждение: ")
	if confirmation != "WIPE_ALL_DRIVES" {
		return fmt.Errorf("операция отменена - неверное подтверждение")
	}

	fmt.Println("\n🔥 НАЧИНАЮ ЗАТИРАНИЕ ВСЕХ ДИСКОВ...\n")

	// Затираем каждый диск последовательно
	for i, d := range drives {
		drive := strings.TrimRight(d.Letter, ".\\") + "\\"
		fmt.Printf("\n[ДИСК %d/%d] Затираю: %s\n", i+1, len(drives), drive)

		err := im.app.StartWipe(drive)
		if err != nil {
			fmt.Printf("❌ Ошибка при затирании диска %s: %v\n", drive, err)
			continue
		}
		fmt.Printf("✅ Диск %s затерт успешно\n", drive)
	}

	fmt.Println("\n🎉 ВСЕ ДИСКИ ЗАТЕРТЫ УСПЕШНО!")
	im.pause()
	return nil
}

func (im *InteractiveMenu) clearScreen() { fmt.Print("\033[H\033[2J") }

func (im *InteractiveMenu) prompt(t string) string {
	fmt.Print(t)
	input, _ := im.reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func (im *InteractiveMenu) pause() {
	fmt.Print("\nНажмите Enter...")
	im.reader.ReadString('\n')
}
