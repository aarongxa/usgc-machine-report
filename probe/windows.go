//go:build windows

package probe

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

func collectOS(r *Report) {
	r.OSName = "Windows (unsupported platform — minimal report)"
	r.Kernel = fmt.Sprintf("%s %s", runtime.GOOS, runtime.GOARCH)
	r.Processor = runtime.GOARCH
	r.Cores = fmt.Sprintf("%d vCPU(s) / 1 Socket(s)", runtime.NumCPU())
	r.NumCPU = runtime.NumCPU()
	r.Hypervisor = "N/A"
	r.CPUFreqGHz = "N/A"
	r.LastLogin = "N/A"
	r.Uptime = "N/A"
	collectDisk(r)
	collectMemWin(r)
	if c := os.Getenv("SSH_CLIENT"); c != "" {
		// rare
	}
}

func collectDisk(r *Report) {
	root := `C:\`
	if h, err := os.Getwd(); err == nil && len(h) >= 3 {
		root = h[:3]
	}
	var freeBytes, totalBytes, totalFree uint64
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceEx := kernel32.NewProc("GetDiskFreeSpaceExW")
	pathPtr, _ := syscall.UTF16PtrFromString(root)
	ret, _, _ := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytes)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFree)),
	)
	if ret == 0 || totalBytes == 0 {
		r.Volume = "N/A"
		return
	}
	used := totalBytes - freeBytes
	r.DiskUsed = used
	r.DiskTotal = totalBytes
	usedGB := float64(used) / (1024 * 1024 * 1024)
	totalGB := float64(totalBytes) / (1024 * 1024 * 1024)
	pct := float64(used) / float64(totalBytes) * 100
	r.Volume = fmt.Sprintf("%.2f/%.2f GB [%.2f%%]", usedGB, totalGB, pct)
}

func collectMemWin(r *Report) {
	type memoryStatusEx struct {
		Length               uint32
		MemoryLoad           uint32
		TotalPhys            uint64
		AvailPhys            uint64
		TotalPageFile        uint64
		AvailPageFile        uint64
		TotalVirtual         uint64
		AvailVirtual         uint64
		AvailExtendedVirtual uint64
	}
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	globalMemoryStatusEx := kernel32.NewProc("GlobalMemoryStatusEx")
	var mem memoryStatusEx
	mem.Length = uint32(unsafe.Sizeof(mem))
	ret, _, _ := globalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&mem)))
	if ret == 0 {
		return
	}
	r.MemTotalKiB = mem.TotalPhys / 1024
	r.MemUsedKiB = (mem.TotalPhys - mem.AvailPhys) / 1024
	_ = time.Second
}
