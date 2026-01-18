package system

// DiskInfo contains information about a disk
type DiskInfo struct {
	Letter     string
	Type       string // HDD/SSD/Unknown
	TotalSize  uint64
	FreeSize   uint64
	UsedSize   uint64
	IsSystem   bool
	IsWritable bool
	Model      string
	Serial     string
	Interface  string
}
