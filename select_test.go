package goselect

import (
	"fmt"
	"io"
	"os"
	"testing"
	"time"
)

type fder interface {
	Fd() uintptr
}

func TestReadWriteSync(t *testing.T) {
	const count = 50
	rrs := []io.Reader{}
	wws := []io.Writer{}
	rFDSet := &FDSet{}
	for i := 0; i < count; i++ {
		rr, ww, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		rrs = append(rrs, rr)
		wws = append(wws, ww)
	}

	go func() {
		time.Sleep(time.Second)
		for i := 0; i < count; i++ {
			fmt.Fprintf(wws[i], "hello %d", i)
			time.Sleep(10 * time.Millisecond)
		}
	}()

	buf := make([]byte, 1024)
	for i := 0; i < count; i++ {
		rFDSet.Zero()
		for i := 0; i < count; i++ {
			rFDSet.Set(rrs[i].(fder).Fd())
		}

		if err := RetrySelect(1024, rFDSet, nil, nil, -1, 10, 10*time.Millisecond); err != nil {
			t.Fatalf("select call failed: %s", err)
		}
		for j := 0; j < count; j++ {
			if rFDSet.IsSet(rrs[j].(fder).Fd()) {
				//				println(i, j)
				if i != j {
					t.Fatalf("unexpected fd ready: %d,expected: %d", j, i)
				}
				_, err := rrs[j].Read(buf)
				if err != nil {
					t.Fatalf("read call failed: %s", err)
				}
			}
		}
	}
}

func TestSelect_readEmptyPipe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		if err := r.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	rFDSet := &FDSet{}
	rFDSet.Set(r.Fd())
	max := r.Fd()
	if err := Select(int(max+1), rFDSet, nil, nil, 10*time.Millisecond); err != nil {
		t.Fatalf("select call failed: %s", err)
	}

	if rFDSet.IsSet(r.Fd()) {
		t.Fatal("Nothing written, the pipe should not have been ready for reading")
	}
}

func TestSelect_readClosedPipe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	// This should make select tell us the read end is ready for reading
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	rFDSet := &FDSet{}
	rFDSet.Set(r.Fd())
	max := r.Fd()
	// "-1" means wait forever. We should return immediately anyway, so that
	// should be fine.
	if err := Select(int(max+1), rFDSet, nil, nil, -1); err != nil {
		t.Fatalf("select call failed: %s", err)
	}

	if !rFDSet.IsSet(r.Fd()) {
		t.Fatal("Closing the write end should have made the read end ready for reading")
	}
}

func TestSelect_readWrittenPipe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		if err := r.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	// This should make select tell us the read end is ready for reading
	if _, err := w.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}

	rFDSet := &FDSet{}
	rFDSet.Set(r.Fd())
	max := r.Fd()
	// "-1" means wait forever. We should return immediately anyway, so that
	// should be fine.
	if err := Select(int(max+1), rFDSet, nil, nil, -1); err != nil {
		t.Fatalf("select call failed: %s", err)
	}

	if !rFDSet.IsSet(r.Fd()) {
		t.Fatal("The pipe has bytes, should have been ready for reading")
	}
}
