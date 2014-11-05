package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"syscall"
	"time"
)

const (
	// NBBY is the amount of bits in a byte
	NBBY = 8
	// NFDBITS is the amount of bits per mask
	NFDBITS = 4 * NBBY
)

// C.__DARWIN_FD_SET
func fdSet(fd uintptr, fds *syscall.FdSet) {
	fds.Bits[fd/NFDBITS] |= int32(1 << (fd % NFDBITS))
}

// C.__DARWIN_FD_CLR
func fdClear(fd uintptr, fds *syscall.FdSet) {
	fds.Bits[fd/NFDBITS] &^= int32(1 << (fd % NFDBITS))
}

// C.__darwin_fd_isset
func fdIsSet(fd uintptr, fds *syscall.FdSet) bool {
	return fds.Bits[fd/NFDBITS]&int32(1<<(fd%NFDBITS)) != 0
}

var nullFdSet = &syscall.FdSet{}

func fdZero(fds *syscall.FdSet) {
	copy(fds.Bits[:], (nullFdSet).Bits[:])
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
	// fmt.Printf("%v\n", rfds.Bits)
	// pretty.Printf("%# v\n", pretty.Diff(rfds.Bits, rfds.Bits))
	for {
		if err := Select(int(r.pipeR.Fd()+1), &rfds, nil, nil, r.Timeout); err != nil {
			return 0, err
		}
		switch {
		case fdIsSet(r.fd, &rfds):
			return r.Reader.Read(b)
		case fdIsSet(r.pipeR.Fd(), &rfds):
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

func test2() error {
	const count = 500
	rrs := []io.Reader{}
	wws := []io.Writer{}
	rfds := &syscall.FdSet{}
	for i := 0; i < count; i++ {
		rr, ww, _ := os.Pipe()
		rrs = append(rrs, rr)
		wws = append(wws, ww)
	}

	go func() {
		time.Sleep(time.Second)
		for i := 0; i < count; i++ {
			fmt.Fprintf(wws[i], "hello %d", i)
			time.Sleep(100 * time.Millisecond)
		}
	}()

	buf := make([]byte, 1024)
	for i := 0; i < count; i++ {
		fdZero(rfds)
		for i := 0; i < count; i++ {
			fdSet(rrs[i].(fder).Fd(), rfds)
		}
		if err := Select(1024, rfds, nil, nil, -1); err != nil {
			return err
		}
		for j := 0; j < count; j++ {
			if fdIsSet(rrs[j].(fder).Fd(), rfds) {
				//				println(i, j)
				if i != j {
					return fmt.Errorf("unexpected fd ready: %d", j)
				}
				n, err := rrs[j].Read(buf)
				if err != nil {
					return err
				}
				fmt.Printf(">%s<\n", buf[:n])
			}
		}
	}
	return nil
}

func test() error {
	rr, ww, err := os.Pipe()
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
			fmt.Printf("-----> %d, %s, %v\n", n, buf[:n], err)
		}
	}()
	for i := 0; i < 2; i++ {
		time.Sleep(1 * time.Second)
		println("interrupt")
		_ = ww
		//ww.Write([]byte("hello"))
		r.Interrupt()
		println("post interrupt")
	}

	time.Sleep(5 * time.Second)
	return nil
}

func main() {
	if err := test2(); err != nil {
		log.Fatal(err)
	}
}
