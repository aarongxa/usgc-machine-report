//go:build !(linux || darwin || freebsd || openbsd || netbsd)

package main

import "os"

func isTerminal(f *os.File) bool {
	return false
}
