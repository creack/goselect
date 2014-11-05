package main

import "syscall"

const (
	// NBBY is the amount of bits in a byte
	NBBY = 8
	// NFDBITS is the amount of bits per mask
	NFDBITS = 4 * NBBY
)

// FDSet wraps syscall.FdSet with convenience methods
type FDSet syscall.FdSet

// Set adds the fd to the set
func (fds *FDSet) Set(fd uintptr) {
	// C.__DARWIN_FD_SET
	fds.Bits[fd/NFDBITS] |= int32(1 << (fd % NFDBITS))
}

// Clear remove the fd from the set
func (fds *FDSet) Clear(fd uintptr) {
	// C.__DARWIN_FD_CLR
	fds.Bits[fd/NFDBITS] &^= int32(1 << (fd % NFDBITS))
}

// IsSet check if the given fd is set
func (fds *FDSet) IsSet(fd uintptr) bool {
	// C.__darwin_fd_isset
	return fds.Bits[fd/NFDBITS]&int32(1<<(fd%NFDBITS)) != 0
}

// Keep a null set to avoid reinstatiation
var nullFdSet = &syscall.FdSet{}

// Zero empties the Set
func (fds *FDSet) Zero() {
	copy(fds.Bits[:], (nullFdSet).Bits[:])
}
