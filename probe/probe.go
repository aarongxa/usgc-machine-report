// Package probe gathers host information without shelling out.
package probe

import (
	"fmt"
	"net"
	"os"
	"os/user"
	"strings"
	"time"
)

// Report holds all fields rendered by the TR-100 machine report.
type Report struct {
	OSName     string
	Kernel     string
	Hostname   string
	MachineIP  string
	ClientIP   string
	DNS        []string
	User       string
	Processor  string
	Cores      string // e.g. "8 vCPU(s) / 1 Socket(s)"
	Hypervisor string
	CPUFreqGHz string
	Load1      float64
	Load5      float64
	Load15     float64
	NumCPU     int // for load bar scale
	Volume     string
	DiskUsed   uint64
	DiskTotal  uint64
	ZFSHealth  string // empty if not ZFS
	MemUsedKiB uint64
	MemTotalKiB uint64
	LastLogin  string
	LastLoginIP string
	Uptime     string
	IMDS       string // optional cloud identity line
	Container  string // soft-fail runtime hint; empty on bare metal
}

// Collect gathers a Report for the current OS. Soft-fails missing fields.
func Collect(opts Options) Report {
	r := Report{
		Hostname:  hostname(),
		User:      currentUser(),
		MachineIP: primaryIP(),
		ClientIP:  "Not connected",
		DNS:       dnsServers(),
		NumCPU:    1,
	}
	collectOS(&r)
	if r.NumCPU < 1 {
		r.NumCPU = 1
	}
	if opts.IMDS {
		if id := probeIMDS(); id != "" {
			r.IMDS = id
		}
	}
	return r
}

// Options controls optional probes.
type Options struct {
	IMDS bool // attempt AWS IMDSv2
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil || h == "" {
		return "Not Defined"
	}
	return h
}

func currentUser() string {
	u, err := user.Current()
	if err != nil {
		return os.Getenv("USER")
	}
	if u.Username != "" {
		return u.Username
	}
	return u.Name
}

func primaryIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "No IP found"
	}
	var v4, v6 string
	for _, iface := range ifaces {
		name := iface.Name
		if name == "lo" || name == "lo0" || strings.HasPrefix(name, "docker") {
			continue
		}
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			if ip4 := ip.To4(); ip4 != nil {
				if v4 == "" {
					v4 = ip4.String()
				}
			} else if v6 == "" {
				v6 = ip.String()
			}
		}
	}
	if v4 != "" {
		return v4
	}
	if v6 != "" {
		return v6
	}
	return "No IP found"
}

func dnsServers() []string {
	data, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return nil
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "nameserver") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			out = append(out, fields[1])
		}
	}
	return out
}

func formatUptime(d time.Duration) string {
	d = d.Round(time.Minute)
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	var parts []string
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%d d", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%d h", hours))
	}
	if mins > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%d m", mins))
	}
	return strings.Join(parts, ", ")
}

func parseOSRelease(path string) (name string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "Unknown"
	}
	var id, version, pretty string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		v = strings.Trim(v, `"'`)
		switch k {
		case "ID":
			id = v
		case "VERSION":
			version = v
		case "PRETTY_NAME":
			pretty = v
		}
	}
	if pretty != "" {
		return pretty
	}
	parts := []string{}
	if id != "" {
		parts = append(parts, titleCase(id))
	}
	if version != "" {
		parts = append(parts, version)
	}
	if len(parts) == 0 {
		return "Unknown"
	}
	return strings.Join(parts, " ")
}


func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
