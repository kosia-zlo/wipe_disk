package system

import "strings"

const (
	// Windows error codes
	ERROR_NOT_READY = 0x15
	ERROR_DISK_FULL = 112
)

func IsWindowsError(err error, code uint32) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())

	switch code {
	case ERROR_DISK_FULL:
		return strings.Contains(msg, "disk full") ||
			strings.Contains(msg, "not enough space") ||
			strings.Contains(msg, "no space")
	case ERROR_NOT_READY:
		return strings.Contains(msg, "not ready") ||
			strings.Contains(msg, "device not ready")
	}
	return false
}
