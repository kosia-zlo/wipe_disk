package wipe

import (
	"crypto/rand"
	"fmt"
	"os"

	"wipedisk_enterprise/internal/logging"
)

// WipeMethod определяет метод заполнения данных
type WipeMethod string

const (
	MethodRandom        WipeMethod = "random"
	MethodZero          WipeMethod = "zero"
	MethodDOD5220       WipeMethod = "dod5220"
	MethodSDeleteCompat WipeMethod = "sdelete-compatible"
)

// FillPattern генерирует паттерн для заполнения в зависимости от метода
func FillPattern(method WipeMethod, pass int, size int) ([]byte, error) {
	switch method {
	case MethodRandom:
		data := make([]byte, size)
		if _, err := rand.Read(data); err != nil {
			return nil, fmt.Errorf("ошибка генерации случайных данных: %w", err)
		}
		return data, nil

	case MethodZero:
		return make([]byte, size), nil

	case MethodDOD5220:
		// DOD 5220.22-M: 3 прохода - случайные, нули, случайные
		switch pass % 3 {
		case 0, 2: // 1-й и 3-й проходы - случайные
			data := make([]byte, size)
			if _, err := rand.Read(data); err != nil {
				return nil, fmt.Errorf("ошибка генерации случайных данных: %w", err)
			}
			return data, nil
		case 1: // 2-й проход - нули
			return make([]byte, size), nil
		}
		// Этот return никогда не будет достигнут, но нужен для компилятора
		return nil, fmt.Errorf("неверный проход для метода DOD5220: %d", pass)

	case MethodSDeleteCompat:
		// SDelete совместимость: 1 проход случайными данными
		data := make([]byte, size)
		if _, err := rand.Read(data); err != nil {
			return nil, fmt.Errorf("ошибка генерации случайных данных: %w", err)
		}
		return data, nil

	default:
		return nil, fmt.Errorf("неизвестный метод затирания: %s", method)
	}
}

// CreateWipeFileWithMethod создает файл с использованием указанного метода
func CreateWipeFileWithMethod(filename string, fileSize uint64, method WipeMethod, pass int, maxSpeedMBps float64, logger *logging.EnterpriseLogger) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	throttledWriter := NewThrottledWriter(file, maxSpeedMBps)

	// Определяем размер чанка в зависимости от метода
	chunkSize := getChunkSizeForMethod(method)

	var written uint64
	for written < fileSize {
		remaining := fileSize - written
		toWrite := uint64(chunkSize)
		if remaining < toWrite {
			toWrite = remaining
		}

		// Генерируем паттерн для заполнения
		pattern, err := FillPattern(method, pass, int(toWrite))
		if err != nil {
			return fmt.Errorf("ошибка генерации паттерна: %w", err)
		}

		// Записываем данные с throttling
		off := 0
		for off < int(toWrite) {
			n, err := throttledWriter.Write(pattern[off:])
			if n > 0 {
				off += n
				written += uint64(n)
			}
			if err != nil {
				return fmt.Errorf("ошибка записи в файл: %w", err)
			}
			if n == 0 {
				return fmt.Errorf("запись вернула 0 байт без ошибки")
			}
		}
	}

	// Синхронизация данных с диском
	if err := throttledWriter.Sync(); err != nil {
		return err
	}

	return nil
}

// getChunkSizeForMethod возвращает оптимальный размер чанка для метода
func getChunkSizeForMethod(method WipeMethod) int {
	switch method {
	case MethodRandom, MethodSDeleteCompat:
		return 16 * 1024 * 1024 // 16MB для случайных данных
	case MethodZero:
		return 32 * 1024 * 1024 // 32MB для нулей (быстрее)
	case MethodDOD5220:
		return 8 * 1024 * 1024 // 8MB для DOD (больше проходов)
	default:
		return 16 * 1024 * 1024
	}
}

// IsSDeleteCompatible проверяет, совместим ли метод с SDelete
func IsSDeleteCompatible(method WipeMethod) bool {
	return method == MethodSDeleteCompat || method == MethodRandom
}

// GetMethodPasses возвращает количество проходов для метода
func GetMethodPasses(method WipeMethod) int {
	switch method {
	case MethodRandom, MethodZero, MethodSDeleteCompat:
		return 1
	case MethodDOD5220:
		return 3
	default:
		return 1
	}
}

// ValidateMethod проверяет корректность метода
func ValidateMethod(method string) (WipeMethod, error) {
	m := WipeMethod(method)
	switch m {
	case MethodRandom, MethodZero, MethodDOD5220, MethodSDeleteCompat:
		return m, nil
	default:
		return "", fmt.Errorf("неподдерживаемый метод затирания: %s", method)
	}
}
