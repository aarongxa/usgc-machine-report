// Package render draws the TR-100 box-drawing machine report.
package render

import (
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/aarongxa/usgc-machine-report/probe"
	"github.com/aarongxa/usgc-machine-report/theme"
)

const (
	minNameLen         = 5
	maxNameLen         = 13
	maxDataLen         = 32
	bordersAndPadding  = 7
	reportTitle        = "UNITED STATES GRAPHICS COMPANY"
	reportSubtitle     = "TR-100 MACHINE REPORT"
)

// BarGraph returns a width-wide bar using █/░ for used/total ratio.
func BarGraph(used, total float64, width int) string {
	if width < 1 {
		width = 1
	}
	if total <= 0 || used <= 0 {
		return strings.Repeat("░", width)
	}
	pct := (used / total) * 100
	if pct > 100 {
		pct = 100
	}
	n := int(math.Round((pct / 100) * float64(width)))
	if n > width {
		n = width
	}
	if n < 0 {
		n = 0
	}
	return strings.Repeat("█", n) + strings.Repeat("░", width-n)
}

// LayoutWidth mirrors the bash CURRENT_LEN calculation (capped data column).
func LayoutWidth(values ...string) int {
	max := 0
	for _, s := range values {
		if n := visibleLen(s); n > max {
			max = n
		}
	}
	if max < maxDataLen {
		return max
	}
	return maxDataLen
}

func visibleLen(s string) int {
	return len([]rune(s))
}

// Report writes the full box-drawing report to w.
func Report(w io.Writer, r probe.Report, th theme.Theme) {
	memUsedGiB := float64(r.MemUsedKiB) / (1024 * 1024)
	memTotalGiB := float64(r.MemTotalKiB) / (1024 * 1024)
	memPct := 0.0
	if r.MemTotalKiB > 0 {
		memPct = float64(r.MemUsedKiB) / float64(r.MemTotalKiB) * 100
	}
	memLine := fmt.Sprintf("%.2f/%.2f GiB [%.2f%%]", memUsedGiB, memTotalGiB, memPct)

	cores := r.NumCPU
	if cores < 1 {
		cores = 1
	}

	// Provisional width from non-bar strings
	candidates := []string{
		reportTitle,
		r.OSName, r.Kernel, r.Hostname, r.MachineIP, r.ClientIP, r.User,
		r.Processor, r.Cores, r.Hypervisor, r.CPUFreqGHz,
		r.Volume, r.ZFSHealth, memLine, r.LastLogin, r.LastLoginIP, r.Uptime, r.IMDS,
	}
	for _, d := range r.DNS {
		candidates = append(candidates, d)
	}
	cur := LayoutWidth(candidates...)
	if cur < 20 {
		cur = 20
	}

	load1 := BarGraph(r.Load1, float64(cores), cur)
	load5 := BarGraph(r.Load5, float64(cores), cur)
	load15 := BarGraph(r.Load15, float64(cores), cur)
	diskBar := BarGraph(float64(r.DiskUsed), float64(r.DiskTotal), cur)
	memBar := BarGraph(float64(r.MemUsedKiB), float64(r.MemTotalKiB), cur)

	b := &builder{w: w, th: th, cur: cur}

	b.header()
	b.centered(reportTitle)
	b.centered(reportSubtitle)
	b.divider("top")
	b.row("OS", r.OSName)
	if r.Container != "" {
		b.row("CONTAINER", r.Container)
	}
	b.row("KERNEL", r.Kernel)
	b.divider("")
	b.row("HOSTNAME", r.Hostname)
	b.row("MACHINE IP", r.MachineIP)
	b.row("CLIENT IP", r.ClientIP)
	for i, dns := range r.DNS {
		b.row(fmt.Sprintf("DNS IP %d", i+1), dns)
	}
	b.row("USER", r.User)
	if r.IMDS != "" {
		b.row("IMDS", r.IMDS)
	}
	b.divider("")
	b.row("PROCESSOR", r.Processor)
	b.row("CORES", r.Cores)
	b.row("HYPERVISOR", r.Hypervisor)
	b.row("CPU FREQ", r.CPUFreqGHz)
	b.rowAccent("LOAD 1m", load1)
	b.rowAccent("LOAD 5m", load5)
	b.rowAccent("LOAD 15m", load15)
	b.divider("")
	b.row("VOLUME", r.Volume)
	b.rowAccent("DISK USAGE", diskBar)
	if r.ZFSHealth != "" {
		b.row("ZFS HEALTH", r.ZFSHealth)
	}
	b.divider("")
	b.row("MEMORY", memLine)
	b.rowAccent("USAGE", memBar)
	b.divider("")
	b.row("LAST LOGIN", r.LastLogin)
	if r.LastLoginIP != "" {
		b.row("", r.LastLoginIP)
	}
	b.row("UPTIME", r.Uptime)
	b.divider("bottom")
}

type builder struct {
	w   io.Writer
	th  theme.Theme
	cur int
}

func (b *builder) c(code, s string) string {
	if code == "" {
		return s
	}
	return code + s + b.th.Reset
}

func (b *builder) header() {
	length := b.cur + maxNameLen + bordersAndPadding
	top := "┌"
	bottom := "├"
	for i := 0; i < length-2; i++ {
		top += "┬"
		bottom += "┴"
	}
	top += "┐"
	bottom += "┤"
	fmt.Fprintln(b.w, b.c(b.th.Border, top))
	fmt.Fprintln(b.w, b.c(b.th.Border, bottom))
}

func (b *builder) centered(text string) {
	maxLen := b.cur + maxNameLen - bordersAndPadding
	totalWidth := maxLen + 12
	textLen := visibleLen(text)
	padLeft := (totalWidth - textLen) / 2
	if padLeft < 0 {
		padLeft = 0
	}
	padRight := totalWidth - textLen - padLeft
	if padRight < 0 {
		padRight = 0
	}
	line := "│" + strings.Repeat(" ", padLeft) + text + strings.Repeat(" ", padRight) + "│"
	// color title text only roughly: wrap whole line border-style then title — keep simple
	fmt.Fprintln(b.w, b.c(b.th.Border, "│")+b.c(b.th.Title, strings.Repeat(" ", padLeft)+text+strings.Repeat(" ", padRight))+b.c(b.th.Border, "│"))
	_ = line
}

func (b *builder) divider(side string) {
	var left, mid, right string
	switch side {
	case "top":
		left, mid, right = "├", "┬", "┤"
	case "bottom":
		left, mid, right = "└", "┴", "┘"
	default:
		left, mid, right = "├", "┼", "┤"
	}
	length := b.cur + maxNameLen + bordersAndPadding
	var sb strings.Builder
	sb.WriteString(left)
	for i := 0; i < length-3; i++ {
		sb.WriteRune('─')
		if i == 14 {
			sb.WriteString(mid)
		}
	}
	sb.WriteString(right)
	fmt.Fprintln(b.w, b.c(b.th.Border, sb.String()))
}

func (b *builder) row(name, data string) {
	b.writeRow(name, data, b.th.Value)
}

func (b *builder) rowAccent(name, data string) {
	b.writeRow(name, data, b.th.Accent)
}

func (b *builder) writeRow(name, data, valueColor string) {
	name = padName(name)
	data = padData(data, b.cur)
	fmt.Fprintf(b.w, "%s %s %s %s %s\n",
		b.c(b.th.Border, "│"),
		b.c(b.th.Label, fmt.Sprintf("%-*s", maxNameLen, name)),
		b.c(b.th.Border, "│"),
		b.c(valueColor, data),
		b.c(b.th.Border, "│"),
	)
}

func padName(name string) string {
	n := visibleLen(name)
	if n < minNameLen {
		return name + strings.Repeat(" ", minNameLen-n)
	}
	if n > maxNameLen {
		runes := []rune(name)
		if len(runes) > maxNameLen-3 {
			return string(runes[:maxNameLen-3]) + "..."
		}
	}
	if n < maxNameLen {
		return name + strings.Repeat(" ", maxNameLen-n)
	}
	return name
}

func padData(data string, width int) string {
	n := visibleLen(data)
	if n >= maxDataLen || n == maxDataLen-1 {
		runes := []rune(data)
		cut := maxDataLen - 3 - 2
		if cut < 1 {
			cut = 1
		}
		if len(runes) > cut {
			return string(runes[:cut]) + "..."
		}
	}
	if n < width {
		return data + strings.Repeat(" ", width-n)
	}
	return data
}
