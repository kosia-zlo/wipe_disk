package wipe

import (
	"time"
)

type WipeOperation struct {
	ID         string
	Disk       string
	Method     string
	Passes     int
	ChunkSize  int64
	Status     string // COMPLETED, PARTIAL, CANCELLED, FAILED
	StartTime  time.Time
	EndTime    *time.Time
	BytesWiped uint64
	SpeedMBps  float64
	Error      string
	Warning    string
}

// SystemDiskPolicy определяет политику безопасности для системного диска
type SystemDiskPolicy struct {
	AllowedPaths    []string
	MaxTempSizeGB   int
	MaxBufferMB     int
	MaxConcurrentIO int
	TimeoutMinutes  int
	ForceWipeSSD    bool
}

// ProgressInfo информация о прогрессе затирания
type ProgressInfo struct {
	BytesWritten uint64
	SpeedMBps    float64
	Percentage   float64
	CurrentFile  string
	Error        error
	Done         bool
}

// WipeResult результат операции затирания
type WipeResult struct {
	Success      bool
	BytesWritten uint64
	Duration     time.Duration
	SpeedMBps    float64
	FilesCreated int
	Error        error
	Cancelled    bool
}
