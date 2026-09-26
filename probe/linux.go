//go:build linux

package probe

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func collectOS(r *Report) {
	r.OSName = parseOSRelease("/etc/os-release")
	r.Kernel = kernelString()
	collectCPU(r)
	collectMem(r)
	collectLoad(r)
	collectDisk(r)
	collectUptime(r)
	collectLastLogin(r)
	collectContainer(r)
}

func kernelString() string {
	var u unix.Utsname
	if err := unix.Uname(&u); err != nil {
		return "Linux"
	}
	sys := unix.ByteSliceToString(u.Sysname[:])
	rel := unix.ByteSliceToString(u.Release[:])
	return strings.TrimSpace(sys + " " + rel)
}

func collectCPU(r *Report) {
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		r.Processor = "Unknown"
		r.Cores = "1 vCPU(s) / 1 Socket(s)"
		r.Hypervisor = "Bare Metal"
		r.NumCPU = 1
		return
	}
	defer f.Close()

	var model string
	var mhz float64
	physicalIDs := map[string]struct{}{}
	coreIDs := map[string]struct{}{}
	processors := 0
	var hypervisor string

	sc := bufio.NewScanner(f)
	var curPhys, curCore string
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			if curPhys != "" {
				physicalIDs[curPhys] = struct{}{}
			}
			if curCore != "" && curPhys != "" {
				coreIDs[curPhys+":"+curCore] = struct{}{}
			}
			curPhys, curCore = "", ""
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		switch k {
		case "processor":
			processors++
		case "model name":
			if model == "" {
				// First four tokens like original bash
				fields := strings.Fields(v)
				n := 4
				if len(fields) < n {
					n = len(fields)
				}
				model = strings.Join(fields[:n], " ")
			}
		case "cpu MHz":
			if mhz == 0 {
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					mhz = f
				}
			}
		case "physical id":
			curPhys = v
		case "core id":
			curCore = v
		case "hypervisor vendor", "Hardware virtualization extension":
			// cpuinfo rarely has hypervisor; leave for later
		}
	}
	if curPhys != "" {
		physicalIDs[curPhys] = struct{}{}
	}
	if curCore != "" && curPhys != "" {
		coreIDs[curPhys+":"+curCore] = struct{}{}
	}

	sockets := len(physicalIDs)
	if sockets == 0 {
		sockets = 1
	}
	coresPerSocket := len(coreIDs) / sockets
	if coresPerSocket == 0 {
		if processors > 0 {
			coresPerSocket = processors / sockets
		} else {
			coresPerSocket = 1
		}
	}
	if processors == 0 {
		processors = coresPerSocket * sockets
	}

	r.Processor = model
	if r.Processor == "" {
		r.Processor = "Unknown"
	}
	r.Cores = fmt.Sprintf("%d vCPU(s) / %d Socket(s)", processors, sockets)
	r.NumCPU = processors
	if mhz > 0 {
		r.CPUFreqGHz = fmt.Sprintf("%.2f GHz", mhz/1000)
	} else {
		r.CPUFreqGHz = "N/A"
	}

	hypervisor = detectHypervisor()
	r.Hypervisor = hypervisor
}

func detectHypervisor() string {
	// DMI product / sysfs without shelling out
	paths := []string{
		"/sys/class/dmi/id/product_name",
		"/sys/class/dmi/id/sys_vendor",
		"/sys/hypervisor/type",
	}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		s := strings.TrimSpace(string(b))
		if s == "" {
			continue
		}
		low := strings.ToLower(s)
		switch {
		case strings.Contains(low, "vmware"):
			return "VMware"
		case strings.Contains(low, "kvm"), strings.Contains(low, "qemu"):
			return "KVM"
		case strings.Contains(low, "microsoft"), strings.Contains(low, "hyper-v"):
			return "Hyper-V"
		case strings.Contains(low, "xen"):
			return "Xen"
		case strings.Contains(low, "virtualbox"):
			return "VirtualBox"
		case strings.Contains(low, "bhyve"):
			return "bhyve"
		case p == "/sys/hypervisor/type":
			return titleCase(s)
		}
	}
	// CPU flags: hypervisor bit often exposed in /proc/cpuinfo
	f, err := os.Open("/proc/cpuinfo")
	if err == nil {
		defer f.Close()
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := sc.Text()
			if strings.HasPrefix(line, "flags") && strings.Contains(line, "hypervisor") {
				return "Virtualized"
			}
		}
	}
	return "Bare Metal"
}

func collectMem(r *Report) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	var total, available uint64
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		v, _ := strconv.ParseUint(fields[1], 10, 64)
		switch fields[0] {
		case "MemTotal:":
			total = v
		case "MemAvailable:":
			available = v
		}
	}
	r.MemTotalKiB = total
	if total >= available {
		r.MemUsedKiB = total - available
	}
}

func collectLoad(r *Report) {
	b, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return
	}
	fields := strings.Fields(string(b))
	if len(fields) >= 3 {
		r.Load1, _ = strconv.ParseFloat(fields[0], 64)
		r.Load5, _ = strconv.ParseFloat(fields[1], 64)
		r.Load15, _ = strconv.ParseFloat(fields[2], 64)
	}
}

func collectDisk(r *Report) {
	// Soft ZFS: if /proc/mounts mentions zfs on /, try reading stats from
	// /proc/spl/kstat/zfs if present; otherwise fall back to Statfs.
	if tryZFS(r) {
		return
	}
	var st syscall.Statfs_t
	if err := syscall.Statfs("/", &st); err != nil {
		r.Volume = "N/A"
		return
	}
	total := st.Blocks * uint64(st.Bsize)
	avail := st.Bavail * uint64(st.Bsize)
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

func tryZFS(r *Report) bool {
	mounts, err := os.ReadFile("/proc/mounts")
	if err != nil || !strings.Contains(string(mounts), " zfs ") {
		return false
	}
	// Without zfs CLI we approximate with Statfs on /; note ZFS presence.
	var st syscall.Statfs_t
	if err := syscall.Statfs("/", &st); err != nil {
		return false
	}
	total := st.Blocks * uint64(st.Bsize)
	avail := st.Bavail * uint64(st.Bsize)
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
	r.ZFSHealth = "ZFS (health N/A without zpool)"
	return true
}

func collectUptime(r *Report) {
	b, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return
	}
	fields := strings.Fields(string(b))
	if len(fields) < 1 {
		return
	}
	secs, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return
	}
	r.Uptime = formatUptime(time.Duration(secs * float64(time.Second)))
}

func collectLastLogin(r *Report) {
	// Soft-fail: read wtmp is complex without CGO; use a gentle default.
	// Prefer lastlog file if present (binary format varies); soft-fail.
	r.LastLogin = "See system logs"
	// SSH_CLIENT / SSH_CONNECTION as best-effort client hint for this session
	if c := os.Getenv("SSH_CLIENT"); c != "" {
		fields := strings.Fields(c)
		if len(fields) >= 1 {
			r.ClientIP = fields[0]
		}
	} else if c := os.Getenv("SSH_CONNECTION"); c != "" {
		fields := strings.Fields(c)
		if len(fields) >= 1 {
			r.ClientIP = fields[0]
		}
	}
}

func collectContainer(r *Report) {
	r.Container = detectContainer()
}

// detectContainer soft-fails a runtime hint. No pid-ns compare (both inodes
// are the guest's from inside a container). Prefer marker files, then cgroup path.
func detectContainer() string {
	if fileExists("/.dockerenv") {
		return "docker"
	}
	if fileExists("/run/.containerenv") {
		return "podman"
	}
	for _, path := range []string{"/proc/1/cgroup", "/proc/self/cgroup"} {
		if hint := containerFromCgroup(path); hint != "" {
			return hint
		}
	}
	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func containerFromCgroup(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return containerHintFromCgroup(string(data))
}

func containerHintFromCgroup(data string) string {
	lower := strings.ToLower(data)
	switch {
	case strings.Contains(lower, "kubepods"):
		return "kubernetes"
	case strings.Contains(lower, "docker"):
		return "docker"
	case strings.Contains(lower, "containerd"):
		return "containerd"
	case strings.Contains(lower, "libpod") || strings.Contains(lower, "podman"):
		return "podman"
	case strings.Contains(lower, "lxc"):
		return "lxc"
	default:
		return ""
	}
}
