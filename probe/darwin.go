//go:build darwin

package probe

import (
	"encoding/binary"
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func collectOS(r *Report) {
	r.OSName = darwinOSName()
	r.Kernel = kernelString()
	collectCPU(r)
	collectMem(r)
	collectLoad(r)
	collectDisk(r)
	collectUptime(r)
	r.LastLogin = "See system logs"
	r.Hypervisor = "Bare Metal"
	if c := os.Getenv("SSH_CLIENT"); c != "" {
		fields := strings.Fields(c)
		if len(fields) >= 1 {
			r.ClientIP = fields[0]
		}
	}
}

func darwinOSName() string {
	ver, err := unix.Sysctl("kern.osproductversion")
	if err != nil || ver == "" {
		ver = "Unknown"
	}
	return "macOS " + ver
}

func kernelString() string {
	var u unix.Utsname
	if err := unix.Uname(&u); err != nil {
		return "Darwin"
	}
	sys := unix.ByteSliceToString(u.Sysname[:])
	rel := unix.ByteSliceToString(u.Release[:])
	return strings.TrimSpace(sys + " " + rel)
}

func collectCPU(r *Report) {
	brand, _ := unix.Sysctl("machdep.cpu.brand_string")
	fields := strings.Fields(brand)
	n := 4
	if len(fields) < n {
		n = len(fields)
	}
	if n > 0 {
		r.Processor = strings.Join(fields[:n], " ")
	} else {
		r.Processor = "Unknown"
	}

	ncpu := runtime.NumCPU()
	r.NumCPU = ncpu
	cores, _ := unix.SysctlUint32("hw.physicalcpu")
	packages, _ := unix.SysctlUint32("hw.packages")
	if packages == 0 {
		packages = 1
	}
	if cores == 0 {
		cores = uint32(ncpu)
	}
	per := cores / packages
	if per == 0 {
		per = uint32(ncpu)
	}
	r.Cores = fmt.Sprintf("%d vCPU(s) / %d Socket(s)", per, packages)

	freq, err := unix.SysctlUint64("hw.cpufrequency")
	if err != nil || freq == 0 {
		r.CPUFreqGHz = "N/A"
	} else {
		r.CPUFreqGHz = fmt.Sprintf("%.2f GHz", float64(freq)/1e9)
	}
}

func collectMem(r *Report) {
	total, err := unix.SysctlUint64("hw.memsize")
	if err != nil {
		return
	}
	r.MemTotalKiB = total / 1024

	pageSize := uint64(os.Getpagesize())
	freePages, err := unix.SysctlUint64("vm.page_free_count")
	if err != nil {
		return
	}
	free := freePages * pageSize
	if total >= free {
		r.MemUsedKiB = (total - free) / 1024
	}
}

func collectLoad(r *Report) {
	// struct loadavg { fixpt_t ldavg[3]; long scale; } via vm.loadavg
	raw, err := unix.SysctlRaw("vm.loadavg")
	if err != nil || len(raw) < 16 {
		return
	}
	// On Darwin, fixpt_t is uint32; scale is typically 65536 (or int64 on 64-bit after 3 uint32s + pad)
	// Layout varies; try common 3x uint32 + uint32/int32 scale, or 3x uint32 + pad + int64.
	var ld [3]uint32
	var scale uint32
	if len(raw) >= 16 {
		ld[0] = binary.LittleEndian.Uint32(raw[0:4])
		ld[1] = binary.LittleEndian.Uint32(raw[4:8])
		ld[2] = binary.LittleEndian.Uint32(raw[8:12])
		scale = binary.LittleEndian.Uint32(raw[12:16])
	}
	if scale == 0 {
		scale = 65536
	}
	r.Load1 = float64(ld[0]) / float64(scale)
	r.Load5 = float64(ld[1]) / float64(scale)
	r.Load15 = float64(ld[2]) / float64(scale)
}

func collectDisk(r *Report) {
	var st syscall.Statfs_t
	if err := syscall.Statfs("/", &st); err != nil {
		r.Volume = "N/A"
		return
	}
	bsize := uint64(st.Bsize)
	total := uint64(st.Blocks) * bsize
	avail := uint64(st.Bavail) * bsize
	used := total - avail
	r.DiskUsed = used
	r.DiskTotal = total
	usedGB := float64(used) / (1024 * 1024 * 1024)
	totalGB := float64(total) / (1024 * 1024 * 1024)
	pct := 0.0
	if total > 0 {
		pct = float64(used) / float64(total) * 100
	}
	r.Volume = fmt.Sprintf("%.2f/%.2f GB [%.2f%%]", usedGB, totalGB, pct)
}

func collectUptime(r *Report) {
	tv, err := unix.SysctlTimeval("kern.boottime")
	if err != nil || tv == nil {
		return
	}
	boot := time.Unix(int64(tv.Sec), int64(tv.Usec)*1000)
	r.Uptime = formatUptime(time.Since(boot))
}
