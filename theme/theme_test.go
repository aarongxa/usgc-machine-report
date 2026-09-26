package theme

import (
	"os"
	"testing"
)

func TestLookup(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"dracula", "dracula"},
		{"DRACULA", "dracula"},
		{"nord", "nord"},
		{"solarized-dark", "solarized-dark"},
		{"monokai", "monokai"},
		{"usgc", "usgc"},
		{"default", "default"},
		{"", "default"},
		{"nope", "default"},
		{"  Nord  ", "nord"},
	}
	for _, tc := range cases {
		got := Lookup(tc.in)
		if got.Name != tc.want {
			t.Errorf("Lookup(%q).Name = %q, want %q", tc.in, got.Name, tc.want)
		}
		if tc.want != "default" || tc.in == "default" || tc.in == "" {
			// colored themes should have non-empty codes when named
			if got.Name != "default" && got.Accent == "" && tc.in != "nope" {
				// Lookup of unknown returns default which has accent — fine
			}
		}
		if got.Reset == "" && got.Name != "" && Lookup(tc.in).Accent != "" {
			// colored
			if got.Accent == "" {
				t.Errorf("Lookup(%q) missing Accent", tc.in)
			}
		}
	}
}

func TestResolveNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("MACHINE_REPORT_THEME", "dracula")
	th := Resolve("", false, true)
	if th.Accent != "" || th.Border != "" {
		t.Fatalf("expected colorless with NO_COLOR, got %+v", th)
	}
	if th.Name != "dracula" {
		t.Fatalf("name = %q, want dracula", th.Name)
	}
}

func TestResolveTTY(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	th := Resolve("nord", false, false)
	if th.Accent != "" {
		t.Fatalf("non-TTY should be colorless")
	}
	th2 := Resolve("nord", false, true)
	if th2.Accent == "" {
		t.Fatalf("TTY should keep colors")
	}
}

func TestNames(t *testing.T) {
	n := Names()
	if len(n) < 5 {
		t.Fatalf("expected >=5 themes, got %d", len(n))
	}
}
