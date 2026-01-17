package wipe

import (
	"math/rand"
	"sync"
)

// BufferPool управляет пулом буферов для оптимизации памяти
type BufferPool struct {
	pools map[int]*sync.Pool
	mu    sync.RWMutex
}

var globalBufferPool = &BufferPool{
	pools: make(map[int]*sync.Pool),
}

// GetBuffer получает буфер из пула или создает новый
func GetBuffer(size int) []byte {
	if size <= 0 {
		return nil
	}

	return globalBufferPool.getBuffer(size)
}

// PutBuffer возвращает буфер в пул
func PutBuffer(buf []byte) {
	if len(buf) == 0 {
		return
	}

	globalBufferPool.putBuffer(buf)
}

// getBuffer получает буфер нужного размера
func (bp *BufferPool) getBuffer(size int) []byte {
	// Находим ближайший размер из существующих пулов
	poolSize := bp.getPoolSize(size)

	bp.mu.RLock()
	pool, exists := bp.pools[poolSize]
	bp.mu.RUnlock()

	if !exists {
		bp.mu.Lock()
		// Double-check
		pool, exists = bp.pools[poolSize]
		if !exists {
			pool = &sync.Pool{
				New: func() interface{} {
					return make([]byte, poolSize)
				},
			}
			bp.pools[poolSize] = pool
		}
		bp.mu.Unlock()
	}

	buf := pool.Get().([]byte)
	return buf[:size] // Возвращаем слайс нужного размера
}

// putBuffer возвращает буфер в соответствующий пул
func (bp *BufferPool) putBuffer(buf []byte) {
	capacity := cap(buf)
	poolSize := bp.getPoolSize(capacity)

	bp.mu.RLock()
	pool, exists := bp.pools[poolSize]
	bp.mu.RUnlock()

	if exists {
		// Сбрасываем буфер перед возвращением в пул
		for i := range buf {
			buf[i] = 0
		}
		pool.Put(buf[:capacity])
	}
}

// getPoolSize определяет размер пула для буфера
func (bp *BufferPool) getPoolSize(size int) int {
	// Стандартные размеры пулов (степени двойки)
	sizes := []int{1024, 4096, 16384, 65536, 262144, 1048576, 4194304, 16777216}

	for _, poolSize := range sizes {
		if size <= poolSize {
			return poolSize
		}
	}

	// Если размер больше максимального, создаем пул точного размера
	return ((size + 4095) / 4096) * 4096 // Округляем до 4KB
}

// FillBufferPattern безопасно заполняет буфер паттерном
func FillBufferPattern(buf []byte, pattern byte) error {
	if len(buf) == 0 {
		return nil
	}

	// Защита от panic в rand
	if len(buf) > 0 {
		for i := range buf {
			buf[i] = pattern
		}
	}

	return nil
}

// FillRandom безопасно заполняет буфер случайными данными
func FillRandom(buf []byte) error {
	if len(buf) == 0 {
		return nil
	}

	// Защита от panic в rand.Int63n
	_, err := rand.Read(buf)
	if err != nil {
		// Fallback к простому заполнению
		for i := range buf {
			buf[i] = byte(rand.Intn(256))
		}
	}

	return nil
}
