//go:build linux

package probe

import "testing"

func TestContainerHintFromCgroup(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"0::/user.slice", ""},
		{"12:pids:/kubepods.slice/kubepods-burstable.slice", "kubernetes"},
		{"1:name=systemd:/docker/abcdef", "docker"},
		{"0::/system.slice/containerd.service", "containerd"},
		{"0::/libpod_parent/libpod-xyz", "podman"},
		{"0::/lxc.payload.ubuntu", "lxc"},
	}
	for _, tc := range cases {
		if got := containerHintFromCgroup(tc.in); got != tc.want {
			t.Errorf("containerHintFromCgroup(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestDetectContainerNoFalsePositiveBare(t *testing.T) {
	// On this box we may or may not be in a container; just ensure it does not panic
	// and returns either "" or a known label.
	got := detectContainer()
	switch got {
	case "", "docker", "podman", "kubernetes", "containerd", "lxc":
	default:
		t.Fatalf("unexpected container hint %q", got)
	}
}
