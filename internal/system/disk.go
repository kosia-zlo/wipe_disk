package system

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// GetDiskInfo gets information about disks via Windows API
func GetDiskInfo(verbose bool) ([]DiskInfo, error) {
	var disks []DiskInfo

	// Get all logical drives
	drives := getLogicalDrives()

	for _, drive := range drives {
		if !isLocalDrive(drive) {
			continue // Skip network drives
		}

		info, err := getDriveInfo(drive)
		if err != nil {
			continue // Skip inaccessible drives
		}

		disks = append(disks, info)
	}

	return disks, nil
}

// GetDiskSpace gets free space information via Windows API
func GetDiskSpace(drive string, verbose bool) (uint64, uint64) {
	// Convert path to UTF16 for Windows API
	drivePath, err := syscall.UTF16PtrFromString(drive)
	if err != nil {
		return 0, 0
	}

	var freeBytesAvailable, totalBytes, freeBytes uint64

	// Call GetDiskFreeSpaceExW with proper uintptr casting
	ret, _, err := procGetDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(drivePath)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&freeBytes)),
	)

	if ret == 0 {
		// API call error
		if verbose {
			fmt.Printf("[ERROR] GetDiskFreeSpaceExW failed for %s: %v\n", drive, err)
		}
		return 0, 0
	}

	return freeBytes, totalBytes
}

// isLocalDrive checks if drive is local (not network)
func isLocalDrive(drive string) bool {
	driveType := windows.GetDriveType(windows.StringToUTF16Ptr(drive))
	return driveType == windows.DRIVE_FIXED
}

// getLogicalDrives gets list of logical drives
func getLogicalDrives() []string {
	var drives []string

	// Use Windows API to get drives
	kernel32, err := syscall.LoadLibrary("kernel32.dll")
	if err != nil {
		// Fallback to simple method
		for c := 'A'; c <= 'Z'; c++ {
			drive := string(c) + ":"
			if _, err := os.Stat(drive + "\\"); err == nil {
				drives = append(drives, drive)
			}
		}
		return drives
	}
	defer syscall.FreeLibrary(kernel32)

	getLogicalDrivesProc, err := syscall.GetProcAddress(kernel32, "GetLogicalDrives")
	if err != nil {
		return drives
	}

	ret, _, _ := syscall.Syscall(uintptr(getLogicalDrivesProc), 0, 0, 0, 0)
	drivesMask := uint32(ret)

	for c := 0; c < 26; c++ {
		if drivesMask&(1<<c) != 0 {
			drive := string('A'+c) + ":"
			drives = append(drives, drive)
		}
	}

	return drives
}

// getDriveInfo gets detailed drive information
func getDriveInfo(drive string) (DiskInfo, error) {
	info := DiskInfo{
		Letter:     drive,
		Type:       "Unknown",
		IsSystem:   isSystemDrive(drive),
		IsWritable: true,
		Model:      "Unknown Model",
		Serial:     "Unknown Serial",
		Interface:  "Unknown Interface",
	}

	// Get free space information
	freeSize, totalSize := GetDiskSpace(drive, false)
	info.FreeSize = freeSize
	info.TotalSize = totalSize

	// Determine disk type (simplified)
	if info.IsSystem {
		info.Type = "SSD" // Assume SSD for system drive
	} else {
		info.Type = "HDD" // Assume HDD for other drives
	}

	// Check write access
	info.IsWritable = checkWriteAccess(drive)

	return info, nil
}

// isSystemDrive checks if drive is system drive
func isSystemDrive(drive string) bool {
	windir := strings.ToUpper(os.Getenv("WINDIR"))
	if windir == "" {
		windir = "C:\\WINDOWS"
	}

	return strings.HasPrefix(windir, strings.ToUpper(drive))
}

// checkWriteAccess checks write access to drive
func checkWriteAccess(drive string) bool {
	testPath := drive + "\\"
	testFile := filepath.Join(testPath, ".wipedisk_write_test")

	file, err := os.Create(testFile)
	if err != nil {
		return false
	}

	file.Close()
	os.Remove(testFile)

	return true
}

// CheckWriteAccess public function for external use
func CheckWriteAccess(drive string) bool {
	return checkWriteAccess(drive)
}

// ValidateDrive validates if drive exists and shows available drives if not
func ValidateDrive(drive string) error {
	if drive == "" {
		return fmt.Errorf("пустой путь к диску")
	}

	// Normalize drive path
	drive = normalizePath(drive)

	// Get all available drives
	availableDrives := getLogicalDrives()

	// Check if requested drive exists
	for _, availableDrive := range availableDrives {
		if normalizePath(availableDrive) == drive {
			// Check if drive is accessible
			if _, err := os.Stat(drive + "\\"); err != nil {
				return fmt.Errorf("диск %s недоступен: %w", drive, err)
			}
			return nil
		}
	}

	// Drive not found, show error with available options
	return fmt.Errorf("ошибка: путь %s недоступен. Пожалуйста, выберите диск из списка доступных: %v",
		drive, availableDrives)
}

// GetAvailableDrives returns list of available local drives with types
func GetAvailableDrives() []DriveInfo {
	var drives []DriveInfo

	// Use Windows API to get drive strings
	buffer := make([]uint16, 256)
	_, err := windows.GetLogicalDriveStrings(uint32(len(buffer)), &buffer[0])
	if err != nil {
		// Fallback to simple method
		for c := 'A'; c <= 'Z'; c++ {
			drive := string(c) + ":"
			if _, err := os.Stat(drive + "\\"); err == nil {
				driveType := windows.GetDriveType(windows.StringToUTF16Ptr(drive))
				if driveType == windows.DRIVE_FIXED || driveType == windows.DRIVE_REMOVABLE {
					drives = append(drives, DriveInfo{
						Letter:   drive,
						Type:     getDriveTypeName(driveType),
						IsSystem: isSystemDrive(drive),
					})
				}
			}
		}
		return drives
	}

	// Parse buffer (null-separated drive strings)
	driveStr := windows.UTF16PtrToString(&buffer[0])
	for _, drive := range strings.Split(driveStr, "\x00") {
		if drive == "" {
			continue
		}

		// Check if drive is accessible
		if _, err := os.Stat(drive + "\\"); err != nil {
			continue
		}

		// Get drive type
		driveType := windows.GetDriveType(windows.StringToUTF16Ptr(drive))
		if driveType != windows.DRIVE_FIXED && driveType != windows.DRIVE_REMOVABLE {
			continue
		}

		// Get free space
		freeSpace, _ := GetDiskSpace(drive, false)

		drives = append(drives, DriveInfo{
			Letter:   drive,
			Type:     getDriveTypeName(driveType),
			IsSystem: isSystemDrive(drive),
			FreeSize: freeSpace,
		})
	}

	return drives
}

// DriveInfo represents information about a drive
type DriveInfo struct {
	Letter   string
	Type     string
	IsSystem bool
	FreeSize uint64
}

// getDriveTypeName converts Windows drive type to readable string
func getDriveTypeName(driveType uint32) string {
	switch driveType {
	case windows.DRIVE_FIXED:
		return "Fixed Drive"
	case windows.DRIVE_REMOVABLE:
		return "Removable Drive"
	case windows.DRIVE_CDROM:
		return "CD-ROM"
	case windows.DRIVE_RAMDISK:
		return "RAM Disk"
	default:
		return "Unknown"
	}
}

// ValidatePath validates and normalizes path
func ValidatePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}

	// Expand environment variables
	expanded := os.ExpandEnv(path)

	// Convert to absolute path
	absPath, err := filepath.Abs(expanded)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Check existence
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return "", fmt.Errorf("path does not exist: %s", absPath)
	}

	return absPath, nil
}

// GetSystemPaths returns system paths
func GetSystemPaths() map[string]string {
	paths := make(map[string]string)

	if temp := os.Getenv("TEMP"); temp != "" {
		paths["TEMP"] = temp
	}

	if tmp := os.Getenv("TMP"); tmp != "" {
		paths["TMP"] = tmp
	}

	if windir := os.Getenv("WINDIR"); windir != "" {
		paths["WINDIR"] = windir
		paths["WINDIR_TEMP"] = filepath.Join(windir, "Temp")
	}

	if localappdata := os.Getenv("LOCALAPPDATA"); localappdata != "" {
		paths["LOCALAPPDATA_TEMP"] = filepath.Join(localappdata, "Temp")
	}

	if userprofile := os.Getenv("USERPROFILE"); userprofile != "" {
		paths["USERPROFILE"] = userprofile
	}

	return paths
}

// GetSafeTempPaths returns safe temporary paths
func GetSafeTempPaths() ([]string, error) {
	var paths []string

	// Add standard temp directories
	if temp := os.Getenv("TEMP"); temp != "" {
		if validated, err := ValidatePath(temp); err == nil {
			paths = append(paths, validated)
		}
	}

	if tmp := os.Getenv("TMP"); tmp != "" {
		if validated, err := ValidatePath(tmp); err == nil {
			paths = append(paths, validated)
		}
	}

	// Add Windows temp directory
	if windir := os.Getenv("WINDIR"); windir != "" {
		windirTemp := filepath.Join(windir, "Temp")
		if validated, err := ValidatePath(windirTemp); err == nil {
			paths = append(paths, validated)
		}
	}

	if len(paths) == 0 {
		return nil, fmt.Errorf("no valid temporary paths found")
	}

	return paths, nil
}

// IsDiskFullError проверяет, является ли ошибка ошибкой "Недостаточно места на диске"
func IsDiskFullError(err error) bool {
	if err == nil {
		return false
	}

	if errno, ok := err.(syscall.Errno); ok && errno == 112 {
		return true
	}

	// Also check for Windows syscall errors
	if pathErr, ok := err.(*os.PathError); ok {
		if errno, ok := pathErr.Err.(syscall.Errno); ok && errno == 112 {
			return true
		}
	}

	return false
}

// IsAdmin checks if current process has administrator privileges
func IsAdmin() bool {
	var sid *windows.SID

	// Create well-known SID for administrators group
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid,
	)
	if err != nil {
		// Fallback: try opening physical drive
		_, fallbackErr := os.Open("\\\\.\\PHYSICALDRIVE0")
		if fallbackErr == nil {
			return true
		}
		return false
	}
	defer windows.FreeSid(sid)

	// Get current process token
	token, err := windows.OpenCurrentProcessToken()
	if err != nil {
		// Fallback: try opening physical drive
		_, fallbackErr := os.Open("\\\\.\\PHYSICALDRIVE0")
		if fallbackErr == nil {
			return true
		}
		return false
	}
	defer token.Close()

	// Check if token is member of administrators group
	member, err := token.IsMember(sid)
	if err != nil {
		// Fallback: try opening physical drive
		_, fallbackErr := os.Open("\\\\.\\PHYSICALDRIVE0")
		if fallbackErr == nil {
			return true
		}
		return false
	}

	return member
}

// Windows API functions for GetDiskFreeSpaceEx
var (
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	procGetDiskFreeSpaceExW = kernel32.NewProc("GetDiskFreeSpaceExW")
)

// GetDiskInfoForPath получает информацию о конкретном диске по пути
func GetDiskInfoForPath(drivePath string) (DiskInfo, error) {
	drivePath = normalizePath(drivePath)

	var freeBytesAvailable, totalBytes, freeBytes uint64

	err := windows.GetDiskFreeSpaceEx(
		windows.StringToUTF16Ptr(drivePath),
		&freeBytesAvailable,
		&totalBytes,
		&freeBytes,
	)
	if err != nil {
		return DiskInfo{}, fmt.Errorf("ошибка получения информации о диске: %w", err)
	}

	// Определение типа диска
	diskType := getDiskType(drivePath)

	// Проверка системного диска
	isSystem := isSystemDisk(drivePath)

	return DiskInfo{
		Letter:     drivePath,
		Type:       diskType,
		TotalSize:  totalBytes,
		FreeSize:   freeBytes,
		UsedSize:   totalBytes - freeBytes,
		IsSystem:   isSystem,
		IsWritable: checkWriteAccess(drivePath),
		Model:      "",
		Serial:     "",
		Interface:  "",
	}, nil
}

// normalizePath нормализует путь к диску
func normalizePath(path string) string {
	if len(path) == 1 {
		return path + ":"
	}
	if len(path) == 2 && path[1] == ':' {
		return path
	}
	if len(path) >= 3 && path[1] == ':' && (path[2] == '\\' || path[2] == '/') {
		return path[:2]
	}
	return path
}

// getDiskType определяет тип диска (HDD/SSD)
func getDiskType(drivePath string) string {
	// В реальной реализации здесь будет определение типа диска
	// через WMI или другие Windows API
	// Пока возвращаем Unknown
	return "Unknown"
}

// isSystemDisk проверяет, является ли диск системным
func isSystemDisk(drivePath string) bool {
	// Получаем путь к системной директории
	sysDir, err := windows.GetSystemDirectory()
	if err != nil {
		return false
	}

	if len(sysDir) >= 2 {
		systemDrive := sysDir[:2]
		return normalizePath(drivePath) == systemDrive
	}

	return false
}
