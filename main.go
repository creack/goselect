package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"syscall"
	"time"
)

type fder interface {
	Fd() uintptr
}

// Reader .
type Reader struct {
	io.Reader
	fd           uintptr
	Timeout      time.Duration
	pipeR, pipeW *os.File
}

// NewReader .
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
	rFDSet := &FDSet{}

	rFDSet.Set(r.fd)
	rFDSet.Set(r.pipeR.Fd())
	for {
		if err := Select(7, rFDSet, nil, nil, r.Timeout); err != nil {
			return 0, err
		}
		switch {
		case rFDSet.IsSet(r.fd):
			return r.Reader.Read(b)
		case rFDSet.IsSet(r.pipeR.Fd()):
			r.pipeR.Read(make([]byte, 1))
			return 0, syscall.EINTR
		default:
			// Timeout
			return 0, syscall.ETIMEDOUT
		}
	}
}

// Interrupt .
func (r *Reader) Interrupt() {
	// Send EOT
	r.pipeW.Write([]byte{4})
}

func test2() error {
	const count = 500
	rrs := []io.Reader{}
	wws := []io.Writer{}
	rFDSet := &FDSet{}
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
		rFDSet.Zero()
		for i := 0; i < count; i++ {
			rFDSet.Set(rrs[i].(fder).Fd())
		}
		if err := Select(1024, rFDSet, nil, nil, -1); err != nil {
			return err
		}
		for j := 0; j < count; j++ {
			if rFDSet.IsSet(rrs[j].(fder).Fd()) {
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
