// TR-100 Machine Report — Go rewrite (cross-OS, static, no shell-outs).
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/aarongxa/usgc-machine-report/probe"
	"github.com/aarongxa/usgc-machine-report/render"
	"github.com/aarongxa/usgc-machine-report/theme"
)

func main() {
	themeFlag := flag.String("theme", "", "color theme: default|usgc|dracula|nord|solarized-dark|monokai (or MACHINE_REPORT_THEME)")
	noColor := flag.Bool("no-color", false, "disable ANSI colors (also respects NO_COLOR)")
	imds := flag.Bool("imds", false, "probe AWS IMDSv2 (soft-fail if unavailable)")
	listThemes := flag.Bool("list-themes", false, "print available theme names and exit")
	flag.Parse()

	if *listThemes {
		for _, n := range theme.Names() {
			fmt.Println(n)
		}
		return
	}

	isTTY := isTerminal(os.Stdout)
	th := theme.Resolve(*themeFlag, *noColor, isTTY)
	rep := probe.Collect(probe.Options{IMDS: *imds})
	render.Report(os.Stdout, rep, th)
}
