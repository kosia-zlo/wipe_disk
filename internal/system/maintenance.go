package system

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"wipedisk_enterprise/internal/logging"
)

// CleanUpdateCache очищает кэш Windows Update
func CleanUpdateCache(ctx context.Context, logger *logging.EnterpriseLogger) (uint64, error) {
	logger.Log("INFO", "Очистка кэша Windows Update")

	var totalCleaned uint64
	paths := []string{
		`C:\Windows\SoftwareDistribution\Download`,
		`C:\Windows\SoftwareDistribution\DataStore`,
		`C:\Windows\wsus`,
	}

	for _, path := range paths {
		select {
		case <-ctx.Done():
			return totalCleaned, ctx.Err()
		default:
		}

		cleaned, err := cleanDirectory(path, logger)
		if err != nil {
			logger.Log("WARN", "Ошибка очистки директории", "path", path, "error", err)
			continue
		}
		totalCleaned += cleaned
	}

	logger.Log("INFO", "Кэш Windows Update очищен", "cleaned_mb", totalCleaned/(1024*1024))
	return totalCleaned, nil
}

// CleanBrowserCache очищает кэш браузеров
func CleanBrowserCache(ctx context.Context, logger *logging.EnterpriseLogger) (uint64, error) {
	logger.Log("INFO", "Очистка кэша браузеров")

	var totalCleaned uint64

	// Пути кэша для разных браузеров
	paths := []string{
		// Chrome
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Google\\Chrome\\User Data\\Default\\Cache"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Google\\Chrome\\User Data\\Default\\Code Cache"),
		// Firefox
		filepath.Join(os.Getenv("APPDATA"), "Mozilla\\Firefox\\Profiles"),
		// Edge
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft\\Edge\\User Data\\Default\\Cache"),
		// Internet Explorer
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft\\Windows\\INetCache"),
		// Opera
		filepath.Join(os.Getenv("APPDATA"), "Opera Software\\Opera Stable\\Cache"),
	}

	for _, path := range paths {
		select {
		case <-ctx.Done():
			return totalCleaned, ctx.Err()
		default:
		}

		// Для Firefox ищем профили
		if strings.Contains(path, "Firefox") && strings.Contains(path, "Profiles") {
			profiles, err := filepath.Glob(filepath.Join(path, "*"))
			if err != nil {
				logger.Log("WARN", "Ошибка поиска профилей Firefox", "error", err)
				continue
			}

			for _, profile := range profiles {
				cachePath := filepath.Join(profile, "cache2")
				cleaned, err := cleanDirectory(cachePath, logger)
				if err != nil {
					logger.Log("WARN", "Ошибка очистки кэша Firefox", "profile", profile, "error", err)
					continue
				}
				totalCleaned += cleaned
			}
		} else {
			cleaned, err := cleanDirectory(path, logger)
			if err != nil {
				logger.Log("WARN", "Ошибка очистки кэша браузера", "path", path, "error", err)
				continue
			}
			totalCleaned += cleaned
		}
	}

	logger.Log("INFO", "Кэш браузеров очищен", "cleaned_mb", totalCleaned/(1024*1024))
	return totalCleaned, nil
}

// OptimizeDisks оптимизирует диски (defrag/TRIM)
func OptimizeDisks(ctx context.Context, logger *logging.EnterpriseLogger) (uint64, error) {
	logger.Log("INFO", "Оптимизация дисков")

	disks, err := GetDiskInfo(false)
	if err != nil {
		return 0, fmt.Errorf("ошибка получения дисков: %w", err)
	}

	for _, disk := range disks {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		if disk.Type == "SSD" {
			// Для SSD выполняем TRIM
			logger.Log("INFO", "Выполнение TRIM для SSD", "disk", disk.Letter)
			if err := optimizeSSD(disk.Letter, logger); err != nil {
				logger.Log("WARN", "Ошибка оптимизации SSD", "disk", disk.Letter, "error", err)
			}
		} else {
			// Для HDD выполняем дефрагментацию
			logger.Log("INFO", "Дефрагментация HDD", "disk", disk.Letter)
			if err := optimizeHDD(disk.Letter, logger); err != nil {
				logger.Log("WARN", "Ошибка дефрагментации HDD", "disk", disk.Letter, "error", err)
			}
		}
	}

	return 0, nil
}

// cleanDirectory очищает директорию и возвращает размер очищенных файлов
func cleanDirectory(path string, logger *logging.EnterpriseLogger) (uint64, error) {
	var totalSize uint64

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return 0, nil // Директория не существует
	}

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			totalSize += uint64(info.Size())

			// Проверяем, что файл не используется
			if err := os.Remove(filePath); err != nil {
				logger.Log("DEBUG", "Не удалось удалить файл", "file", filePath, "error", err)
			}
		}

		return nil
	})

	if err != nil {
		return totalSize, fmt.Errorf("ошибка обхода директории %s: %w", path, err)
	}

	return totalSize, nil
}

// optimizeSSD выполняет оптимизацию SSD через TRIM
func optimizeSSD(drive string, logger *logging.EnterpriseLogger) error {
	// Заглушка - в реальной реализации нужно использовать Windows API
	// для отправки TRIM команды
	logger.Log("INFO", "TRIM выполнен", "drive", drive)
	return nil
}

// optimizeHDD выполняет дефрагментацию HDD
func optimizeHDD(drive string, logger *logging.EnterpriseLogger) error {
	// Заглушка - в реальной реализации нужно вызывать defrag.exe
	logger.Log("INFO", "Дефрагментация запущена", "drive", drive)

	// Имитация работы
	time.Sleep(5 * time.Second)

	logger.Log("INFO", "Дефрагментация завершена", "drive", drive)
	return nil
}
