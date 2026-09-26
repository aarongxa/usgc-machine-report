//go:build linux

package probe

import (
	"syscall"
)

func imdsDialControl(network, address string, c syscall.RawConn) error {
	var sockErr error
	err := c.Control(func(fd uintptr) {
		// Hop limit 1 for IMDSv2 SSRF mitigation
		sockErr = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TTL, 1)
		_ = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IPV6, syscall.IPV6_UNICAST_HOPS, 1)
	})
	if err != nil {
		return err
	}
	return sockErr
}
