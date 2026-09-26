//go:build !linux && !darwin

package main

// Non-unix: treat as non-TTY for color default (windows stub path).
const ioctlReadTermios = 0
