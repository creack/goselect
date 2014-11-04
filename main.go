package main

import (
	"fmt"
	"io"
	"os"
	"syscall"
	"time"
)

// C.FD_SET
func fdSet(fd uintptr, fds *syscall.FdSet) {
	idx := int(fd) / (syscall.FD_SETSIZE / len(fds.Bits)) % len(fds.Bits)
	pos := int(fd) % (syscall.FD_SETSIZE / len(fds.Bits))
	fds.Bits[idx] = 1 << uint(pos)
}

// C.FD_ISSET
func fdIsset(fd uintptr, fds *syscall.FdSet) bool {
	idx := int(fd) / (syscall.FD_SETSIZE / len(fds.Bits)) % len(fds.Bits)
	pos := int(fd) % (syscall.FD_SETSIZE / len(fds.Bits))
	return fds.Bits[idx]&(1<<uint(pos)) != 0
}

type fder interface {
	Fd() uintptr
}

type reader struct {
	io.Reader
	fd           uintptr
	Timeout      time.Duration
	pipeR, pipeW *os.File
}

func NewReader(r io.Reader) (*reader, error) {
	fder, ok := r.(fder)
	if !ok {
		return nil, fmt.Errorf("Can't create a reader with no underlying FD")
	}
	rr, ww, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	return &reader{
		Reader: r,
		fd:     fder.Fd(),
	}, nil
}

func (r *reader) read(b []byte) (int, error) {
	var (
		rfds    syscall.FdSet
		timeout syscall.Timeval
	)

	fdSet(r.fd, &rfds)

	timeout.Sec = r.Timeout.Nanoseconds() / 1E9
	timeout.Usec = int32((r.Timeout.Nanoseconds() % 1E9) / 1E3)

	if err := syscall.Select(int(r.fd+1), &rfds, nil, nil, &timeout); err != nil {
		return 0, err
	}
	if fdIsset(r.fd, &rfds) {
		return r.Reader.Read(b)
	}
	// Timeout
	return 0, fmt.Errorf("modbus: read timeout after %s", r.Timeout.String())
}

func main() {
}
