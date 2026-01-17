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

// DiskInfo contains information about a disk
type DiskInfo struct {
	Letter     string
	Type       string // HDD/SSD/Unknown
	TotalSize  uint64
	FreeSize   uint64
	IsSystem   bool
	IsWritable bool
	Model      string
	Serial     string
	Interface  string
}

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

	// Call GetDiskFreeSpaceExW
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

// Windows API functions for GetDiskFreeSpaceEx
var (
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	procGetDiskFreeSpaceExW = kernel32.NewProc("GetDiskFreeSpaceExW")
)
