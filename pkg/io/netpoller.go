package io

import (
	"os-simulator/pkg/visualizer"
	"strings"
	"sync"
	"time"
)

// NetpollerGoroutine represents a Goroutine parked waiting for network I/O
type NetpollerGoroutine struct {
	GID     int
	FD      int
	Blocked time.Time
	WakeCh  chan struct{}
}

// GoNetpoller simulates Go's Runtime Network Poller
// How Go converts blocking network I/O into non-blocking epoll under the hood:
// 1. User writes `conn.Read(buf)` (looks synchronous).
// 2. Go runtime issues non-blocking `read()` system call (`O_NONBLOCK`).
// 3. If data is ready: returns immediately.
// 4. If `EAGAIN`/`EWOULDBLOCK`: Go runtime registers socket with runtime epoll/kqueue instance.
// 5. Calling Goroutine changes state from `_Grunning` to `_Gwaiting` (parks).
// 6. The OS Thread `M` is NOT blocked! `M` immediately picks up another runnable Goroutine `G`.
// 7. Background `sysmon` thread or idle `M` polls epoll via `netpoll()`.
// 8. When epoll returns ready event, parked Goroutine is moved to `_Grunnable` on a `P`'s local queue!
type GoNetpoller struct {
	EpollInstance *Epoll
	ParkedGs      map[int]*NetpollerGoroutine
	TotalParked   int
	TotalWoken    int
	mu            sync.Mutex
}

func NewGoNetpoller() *GoNetpoller {
	return &GoNetpoller{
		EpollInstance: NewEpoll(),
		ParkedGs:      make(map[int]*NetpollerGoroutine),
	}
}

// ReadNonBlocking simulates Go's net.Conn Read
func (np *GoNetpoller) ReadNonBlocking(gid int, fd *FileDescriptor) ([]byte, bool) {
	// If data already present in socket buffer
	if fd.IsReady && len(fd.DataBuffer) > 0 {
		data := fd.DataBuffer
		fd.DataBuffer = nil
		fd.IsReady = false
		return data, true
	}

	// EAGAIN / EWOULDBLOCK: Park Goroutine on Netpoller
	np.mu.Lock()
	np.TotalParked++
	wakeCh := make(chan struct{})
	np.ParkedGs[fd.FD] = &NetpollerGoroutine{
		GID:     gid,
		FD:      fd.FD,
		Blocked: time.Now(),
		WakeCh:  wakeCh,
	}
	np.mu.Unlock()

	// Register FD in Epoll
	np.EpollInstance.EpollCtl(EPOLL_CTL_ADD, fd, &EpollEvent{
		Events: EPOLLIN,
		FD:     fd,
	})

	return nil, false // Indicates G must park
}

// PollAndWakeup simulates Go runtime `netpoll()` execution by background sysmon
func (np *GoNetpoller) PollAndWakeup() []*NetpollerGoroutine {
	events, _ := np.EpollInstance.EpollWait(64, 10*time.Millisecond)

	np.mu.Lock()
	defer np.mu.Unlock()

	wokenGs := make([]*NetpollerGoroutine, 0)
	for _, ev := range events {
		if parked, exists := np.ParkedGs[ev.FD.FD]; exists {
			np.TotalWoken++
			wokenGs = append(wokenGs, parked)
			delete(np.ParkedGs, ev.FD.FD)
			np.EpollInstance.EpollCtl(EPOLL_CTL_DEL, ev.FD, nil)
			close(parked.WakeCh) // Wake up parked Goroutine
		}
	}

	return wokenGs
}

// RenderNetpollerExplanation visualizes the beauty of Go's Netpoller
func (np *GoNetpoller) RenderNetpollerExplanation() string {
	var sb strings.Builder
	sb.WriteString(visualizer.SubHeader("Go Runtime Netpoller: Epoll + Goroutines Integration"))

	lines := []string{
		"THE BRILLIANCE OF GO's NETPOLLER ARCHITECTURE:",
		"• In C/C++ or traditional Java:",
		"  - Developers must write complex asynchronous callback state machines or event loops (like epoll_wait directly).",
		"  - Or allocate 1 OS thread per connection (10,000 threads = crash).",
		"• In Go:",
		"  - You write clean, sequential, synchronous code: 'data, err := conn.Read()'",
		"  - Runtime Netpoller catches 'EAGAIN' and parks the G into kernel epoll.",
		"  - The OS thread M is instantly freed to run other user Goroutines!",
		"  - When network data arrives, sysmon / epoll wakes the G and puts it in P's queue.",
		"• Zero blocking OS threads, zero memory waste, infinite concurrency scalability!",
	}

	sb.WriteString(visualizer.Box("How Go Dominates High-Concurrency Network Systems", lines))
	return sb.String()
}
