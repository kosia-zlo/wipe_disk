package system

import "os"

// GetSystemDrive возвращает системный диск (C:, D:, и т.д.)
func GetSystemDrive() string {
	// Получаем путь к системной директории
	windir := os.Getenv("WINDIR")
	if windir == "" {
		return "C:" // Fallback
	}

	// Извлекаем букву диска
	if len(windir) >= 2 {
		return windir[:2]
	}

	return "C:" // Fallback
}

// getSystemDrive возвращает системный диск (C:, D:, и т.д.) - для внутреннего использования
func getSystemDrive() string {
	return GetSystemDrive()
}
