package render

import (
	"strings"
	"testing"
)

func TestBarGraph(t *testing.T) {
	cases := []struct {
		used, total float64
		width       int
		wantFilled  int
		wantEmpty   int
	}{
		{0, 100, 10, 0, 10},
		{50, 100, 10, 5, 5},
		{100, 100, 10, 10, 0},
		{150, 100, 10, 10, 0}, // clamp
		{25, 100, 8, 2, 6},
		{1, 0, 5, 0, 5}, // total 0 → empty
	}
	for _, tc := range cases {
		got := BarGraph(tc.used, tc.total, tc.width)
		if visibleLen(got) != tc.width {
			t.Errorf("BarGraph(%v,%v,%d) len=%d want %d (%q)", tc.used, tc.total, tc.width, visibleLen(got), tc.width, got)
		}
		filled := strings.Count(got, "█")
		empty := strings.Count(got, "░")
		if filled != tc.wantFilled || empty != tc.wantEmpty {
			t.Errorf("BarGraph(%v,%v,%d) = %q filled=%d empty=%d want %d/%d",
				tc.used, tc.total, tc.width, got, filled, empty, tc.wantFilled, tc.wantEmpty)
		}
	}
}

func TestLayoutWidth(t *testing.T) {
	if got := LayoutWidth("hi", "hello"); got != 5 {
		t.Errorf("LayoutWidth = %d want 5", got)
	}
	long := strings.Repeat("a", 100)
	if got := LayoutWidth(long); got != maxDataLen {
		t.Errorf("LayoutWidth capped = %d want %d", got, maxDataLen)
	}
}

func TestPadName(t *testing.T) {
	if got := padName("OS"); visibleLen(got) < minNameLen {
		t.Errorf("padName short = %q", got)
	}
	long := strings.Repeat("X", 20)
	got := padName(long)
	if visibleLen(got) != maxNameLen {
		t.Errorf("padName long len=%d want %d (%q)", visibleLen(got), maxNameLen, got)
	}
	if !strings.HasSuffix(got, "...") {
		t.Errorf("padName long should truncate with ...: %q", got)
	}
}
