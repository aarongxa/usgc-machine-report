package probe

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestParseOSReleaseFixture(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "os-release")
	content := `NAME="Debian GNU/Linux"
ID=debian
VERSION="12 (bookworm)"
VERSION_CODENAME=bookworm
PRETTY_NAME="Debian GNU/Linux 12 (bookworm)"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got := parseOSRelease(path)
	want := "Debian GNU/Linux 12 (bookworm)"
	if got != want {
		t.Fatalf("parseOSRelease = %q, want %q", got, want)
	}
}

func TestFormatUptime(t *testing.T) {
	got := formatUptime(0)
	if got == "" {
		t.Fatal("empty uptime")
	}
}

func TestCollectSmoke(t *testing.T) {
	r := Collect(Options{})
	if r.Hostname == "" {
		t.Fatal("hostname empty")
	}
	if runtime.GOOS == "linux" {
		if r.Kernel == "" {
			t.Fatal("kernel empty on linux")
		}
	}
}

func TestTitleCase(t *testing.T) {
	if titleCase("debian") != "Debian" {
		t.Fatal(titleCase("debian"))
	}
	if titleCase("") != "" {
		t.Fatal("empty")
	}
}
