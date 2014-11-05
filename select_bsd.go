// +build !linux

package main

import (
	"syscall"
	"time"
)

func Select(max int, r, w, e *FDSet, timeout time.Duration) error {
	var timeval *syscall.Timeval
	if timeout >= 0 {
		t := syscall.NsecToTimeval(timeout.Nanoseconds())
		timeval = &t
	}
	return syscall.Select(max+1, (*syscall.FdSet)(r), (*syscall.FdSet)(w), (*syscall.FdSet)(e), timeval)
}
