//go:build !linux

package probe

import "syscall"

func imdsDialControl(network, address string, c syscall.RawConn) error {
	return nil
}
