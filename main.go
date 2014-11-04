package main

import (
	"fmt"
	"io"
	"log"
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

type Reader struct {
	io.Reader
	fd           uintptr
	Timeout      time.Duration
	pipeR, pipeW *os.File
}

func NewReader(r io.Reader) (*Reader, error) {
	fder, ok := r.(fder)
	if !ok {
		return nil, fmt.Errorf("Can't create a reader with no underlying FD")
	}
	rr, ww, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	return &Reader{
		Timeout: -1,
		Reader:  r,
		fd:      fder.Fd(),
		pipeR:   rr,
		pipeW:   ww,
	}, nil
}

func (r *Reader) Read(b []byte) (int, error) {
	var (
		rfds syscall.FdSet
	)

	fdSet(r.fd, &rfds)
	fdSet(r.pipeR.Fd(), &rfds)

	for {
		if err := Select(int(r.pipeR.Fd()+1), &rfds, nil, nil, r.Timeout); err != nil {
			return 0, err
		}
		switch {
		case fdIsset(r.fd, &rfds):
			return r.Reader.Read(b)
		case fdIsset(r.pipeR.Fd(), &rfds):
			r.pipeR.Read(make([]byte, 1))
			return 0, syscall.EINTR
		default:
			// Timeout
			return 0, syscall.ETIMEDOUT
		}
	}
}

func (r *Reader) Interrupt() {
	// Send EOT
	r.pipeW.Write([]byte{4})
}

func test() error {
	rr, _, err := os.Pipe()
	if err != nil {
		return err
	}
	r, err := NewReader(rr)
	if err != nil {
		return err
	}

	go func() {
		for {
			buf := make([]byte, 1024)
			println("Before read")
			n, err := r.Read(buf)
			fmt.Printf("-----> %d, %s\n", n, err)
		}
	}()
	for i := 0; i < 2; i++ {
		time.Sleep(1 * time.Second)
		println("interrupt")
		r.Interrupt()
		println("post interrupt")
	}

	time.Sleep(5 * time.Second)
	return nil
}

func main() {
	if err := test(); err != nil {
		log.Fatal(err)
	}
}
