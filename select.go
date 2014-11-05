package goselect

import (
	"syscall"
	"time"
)

const maxRetry = 10

// Select wraps syscall.Select with Go types
func Select(n int, r, w, e *FDSet, timeout time.Duration) error {
	var timeval *syscall.Timeval
	if timeout >= 0 {
		t := syscall.NsecToTimeval(timeout.Nanoseconds())
		timeval = &t
	}

	retry := 0
retry:
	err := sysSelect(n, r, w, e, timeval)
	if err == syscall.EINTR {
		if retry < maxRetry {
			time.Sleep(10 * time.Millisecond)
			retry++
			goto retry
		}
	}
	return err
}
