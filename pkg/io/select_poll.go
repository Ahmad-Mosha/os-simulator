package io

import (
	"fmt"
	"time"
)

// SelectPoll simulates the classic select() and poll() syscalls
// Monitors N descriptors by linearly iterating through all of them (O(N) complexity)
type SelectPoll struct {
	TotalChecks int
}

func NewSelectPoll() *SelectPoll {
	return &SelectPoll{}
}

// Poll monitors all provided file descriptors and returns the list of ready descriptors
// Demonstrates O(N) linear scanning overhead
func (sp *SelectPoll) Poll(fds []*FileDescriptor) (ready []*FileDescriptor, scannedCount int, elapsed time.Duration) {
	start := time.Now()
	sp.TotalChecks++
	scannedCount = len(fds)

	ready = make([]*FileDescriptor, 0)
	// O(N) linear iteration over all file descriptors in user space & kernel
	for _, fd := range fds {
		if fd.IsReady {
			ready = append(ready, fd)
		}
	}

	elapsed = time.Since(start)
	return ready, scannedCount, elapsed
}

func (sp *SelectPoll) ExplainSelectOverhead(totalFDs int) string {
	return fmt.Sprintf(
		"SELECT/POLL SCALING BOTTLENECK (O(N)):\n"+
			"• To monitor %d descriptors, user space must copy an entire %d-element array/bitmap into kernel space.\n"+
			"• Kernel iterates through ALL %d descriptors one by one to check status.\n"+
			"• Even if only 1 descriptor has activity, CPU does %d checks on every call -> Destroys scalability at C10K!",
		totalFDs, totalFDs, totalFDs, totalFDs,
	)
}
