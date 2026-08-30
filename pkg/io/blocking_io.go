package io

import (
	"fmt"
	"time"
)

// FileDescriptor simulates a network socket or file
type FileDescriptor struct {
	FD         int
	Name       string
	DataBuffer []byte
	IsReady    bool
	IsNonBlock bool
}

// BlockingIO simulates synchronous blocking I/O
// Thread blocks on read() until data arrives from network/disk
type BlockingIO struct {
	BlockedThreads int
	TotalReads     int
}

func NewBlockingIO() *BlockingIO {
	return &BlockingIO{}
}

// Read blocks the calling goroutine until data is available on the descriptor
func (bio *BlockingIO) Read(fd *FileDescriptor, delay time.Duration) ([]byte, time.Duration) {
	start := time.Now()
	bio.TotalReads++
	bio.BlockedThreads++

	// Simulate hardware / network packet arrival delay
	time.Sleep(delay)

	bio.BlockedThreads--
	elapsed := time.Since(start)

	if len(fd.DataBuffer) == 0 {
		return []byte(fmt.Sprintf("data-from-fd-%d", fd.FD)), elapsed
	}
	return fd.DataBuffer, elapsed
}
