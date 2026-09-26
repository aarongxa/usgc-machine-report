# TR-100 Machine Report (Go)

SKU: TR-100 — Go rewrite of [usgraphics/usgc-machine-report](https://github.com/usgraphics/usgc-machine-report).

Cross-OS (Linux + Darwin) static binary that prints a login banner in the original box-drawing style. **Never shells out** — probes `/proc`, `sysctl`/`uname`, `Statfs`, and `net.Interfaces` only. Optional soft-fail ZFS detection and AWS IMDSv2.

Module: `github.com/aarongxa/usgc-machine-report`

## Build

Requires Go 1.22+ (tested with Go 1.24).

```bash
cd /path/to/usgc-machine-report
go mod tidy
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o machine-report .
CGO_ENABLED=0 go test ./...
```

Or: `make build && make test`

Cross-compile: `make dist`

## Run

```bash
./machine-report
./machine-report --theme=dracula
./machine-report --no-color
./machine-report --imds
./machine-report --list-themes
```

Environment:

| Variable | Effect |
|----------|--------|
| `MACHINE_REPORT_THEME` | Theme name (overridden by `--theme`) |
| `NO_COLOR` | Disable ANSI colors if set (any value) |

Colors default on when stdout is a TTY and `NO_COLOR` is unset.

Themes: `default`, `usgc`, `dracula`, `nord`, `solarized-dark`, `monokai`.

## Install (per-user, recommended)

Drop the binary into `~/.local/bin` and gate it behind an interactive-shell check. **Default is not** `pam_motd` / `update-motd.d`.

```bash
# one-liner (run from the build dir after `make build`)
mkdir -p ~/.local/bin
cp -f machine-report ~/.local/bin/
chmod +x ~/.local/bin/machine-report

# bash
grep -q 'machine-report' ~/.bashrc 2>/dev/null || cat >> ~/.bashrc <<'EOF'

# TR-100 Machine Report (interactive only)
if [[ $- == *i* ]] && command -v machine-report >/dev/null 2>&1; then
  machine-report
fi
EOF

# zsh
grep -q 'machine-report' ~/.zshrc 2>/dev/null || cat >> ~/.zshrc <<'EOF'

# TR-100 Machine Report (interactive only)
if [[ -o interactive ]] && command -v machine-report >/dev/null 2>&1; then
  machine-report
fi
EOF
```

Ensure `~/.local/bin` is on your `PATH`.

### System-wide (opt-in only)

Not the default. If you intentionally want every login, you may place a wrapper under `/etc/profile.d/` or a pam_motd fragment — document and review carefully; prefer the per-user interactive guard above.

## Layout (packages)

```
.
├── main.go              # flags, theme resolve, collect + render
├── termios_*.go         # TTY ioctl constants (build-tagged)
├── tty_*.go             # isTerminal (unix vs stub)
├── probe/
│   ├── probe.go         # Report struct, shared helpers
│   ├── linux.go         # /proc, Statfs, Uname
│   ├── darwin.go        # sysctl, Getloadavg, Statfs
│   ├── windows.go       # minimal stub
│   └── imds.go          # optional IMDSv2 HTTP
├── theme/               # embedded ANSI palettes
└── render/              # box-drawing + bar graphs
```

## Constraints

- `CGO_ENABLED=0` static binary
- No shell-outs (`lscpu`, `zfs`, `docker`, `kubectl`, `aws`, `hostname` binary, etc.)
- `flag` only (no cobra)
- Allowed extra module: `golang.org/x/sys`

## License

Inspired by U.S. Graphics Company TR-100 (BSD-3-Clause). This Go port is for Aaron G; add a LICENSE file before publishing.
