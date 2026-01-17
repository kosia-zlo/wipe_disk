package system

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"wipedisk_enterprise/internal/logging"
)

// Очистка временных файлов
func CleanTempFiles(ctx context.Context, logger *logging.EnterpriseLogger, dryRun bool) error {
	logger.Log("INFO", "Начало очистки временных файлов")

	paths := []string{
		os.TempDir(),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Temp"),
		`C:\Windows\Temp`,
		`C:\Windows\Prefetch`,
	}

	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if info.IsDir() {
				return nil
			}

			if isExcludedPath(p) {
				return nil
			}

			if dryRun {
				logger.Log("INFO", "DRY RUN: файл удален", "path", p, "size", info.Size())
				return nil
			}

			err = os.Remove(p)
			if err != nil {
				logger.Log("WARN", "Ошибка удаления файла", "path", p, "error", err.Error())
			} else {
				logger.Log("DEBUG", "Файл удален", "path", p, "size", info.Size())
			}

			return nil
		})

		if err != nil {
			logger.Log("ERROR", "Ошибка очистки пути", "path", path, "error", err.Error())
		}
	}

	logger.Log("INFO", "Очистка временных файлов завершена")
	return nil
}

func isExcludedPath(path string) bool {
	excludes := []string{
		".exe", ".dll", ".sys", ".bat", ".cmd",
		"pagefile.sys", "hiberfil.sys", "swapfile.sys",
	}

	for _, exclude := range excludes {
		if strings.HasSuffix(strings.ToLower(path), strings.ToLower(exclude)) {
			return true
		}
	}

	return false
}
