// +build !darwin,!freebsd

package main

import (
	"syscall"
	"time"
)

func Select(n int, r, w, e *syscall.FdSet, timeout time.Duration) error {
	var timeval *syscall.Timeval
	if timeout >= 0 {
		t := syscall.NsecToTimeval(timeout.Nanoseconds())
		timeval = &t
	}
	_, err := syscall.Select(n, r, w, e, timeval)
	return err
}
