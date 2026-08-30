package io

import (
	"fmt"
	"os-simulator/pkg/visualizer"
	"strings"
	"sync"
	"time"
)

// EpollEventType bitmask flags
type EpollEventType uint32

const (
	EPOLLIN      EpollEventType = 0x001 // Ready for reading
	EPOLLOUT     EpollEventType = 0x002 // Ready for writing
	EPOLLET      EpollEventType = 0x004 // Edge-Triggered mode
	EPOLLONESHOT EpollEventType = 0x008 // One-shot notification
)

// EpollEvent represents an event registered or returned by epoll
type EpollEvent struct {
	Events EpollEventType
	FD     *FileDescriptor
}

// EpollCtlOp defines operations for epoll_ctl
type EpollCtlOp int

const (
	EPOLL_CTL_ADD EpollCtlOp = 1
	EPOLL_CTL_MOD EpollCtlOp = 2
	EPOLL_CTL_DEL EpollCtlOp = 3
)

// Epoll simulates Linux epoll / BSD kqueue event notification mechanism
// Architecture:
// 1. Interest List: Red-Black tree / Map storing registered FDs (O(log N) or O(1) modifications)
// 2. Ready List: Doubly-linked list of ready FDs populated by device driver interrupts (O(1) retrieval)
type Epoll struct {
	InterestList map[int]*EpollEvent // Registered FDs
	ReadyList    []*EpollEvent       // FDs with active events
	mu           sync.Mutex
	wakeCh       chan struct{}
}

func NewEpoll() *Epoll {
	return &Epoll{
		InterestList: make(map[int]*EpollEvent),
		ReadyList:    make([]*EpollEvent, 0),
		wakeCh:       make(chan struct{}, 1),
	}
}

// EpollCtl simulates epoll_ctl() syscall: ADD, MOD, DEL
func (ep *Epoll) EpollCtl(op EpollCtlOp, fd *FileDescriptor, event *EpollEvent) error {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	switch op {
	case EPOLL_CTL_ADD:
		if _, exists := ep.InterestList[fd.FD]; exists {
			return fmt.Errorf("EEXIST: file descriptor %d already in epoll interest list", fd.FD)
		}
		ep.InterestList[fd.FD] = event

	case EPOLL_CTL_MOD:
		if _, exists := ep.InterestList[fd.FD]; !exists {
			return fmt.Errorf("ENOENT: file descriptor %d not in epoll interest list", fd.FD)
		}
		ep.InterestList[fd.FD] = event

	case EPOLL_CTL_DEL:
		if _, exists := ep.InterestList[fd.FD]; !exists {
			return fmt.Errorf("ENOENT: file descriptor %d not in epoll interest list", fd.FD)
		}
		delete(ep.InterestList, fd.FD)
	}

	return nil
}

// NotifyDeviceInterrupt simulates hardware interrupt / NIC driver notifying kernel of incoming packet
func (ep *Epoll) NotifyDeviceInterrupt(fd *FileDescriptor, events EpollEventType) {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	if _, exists := ep.InterestList[fd.FD]; exists {
		fd.IsReady = true
		readyEv := &EpollEvent{
			Events: events,
			FD:     fd,
		}
		ep.ReadyList = append(ep.ReadyList, readyEv)

		// Wake up epoll_wait if sleeping
		select {
		case ep.wakeCh <- struct{}{}:
		default:
		}
	}
}

// EpollWait simulates epoll_wait() syscall: returns immediately active events in O(1)
func (ep *Epoll) EpollWait(maxEvents int, timeout time.Duration) ([]*EpollEvent, error) {
	ep.mu.Lock()
	if len(ep.ReadyList) > 0 {
		count := len(ep.ReadyList)
		if count > maxEvents {
			count = maxEvents
		}
		returned := ep.ReadyList[:count]
		ep.ReadyList = ep.ReadyList[count:]
		ep.mu.Unlock()
		return returned, nil
	}
	ep.mu.Unlock()

	// Wait for event or timeout
	select {
	case <-ep.wakeCh:
		ep.mu.Lock()
		count := len(ep.ReadyList)
		if count > maxEvents {
			count = maxEvents
		}
		returned := ep.ReadyList[:count]
		ep.ReadyList = ep.ReadyList[count:]
		ep.mu.Unlock()
		return returned, nil

	case <-time.After(timeout):
		return []*EpollEvent{}, nil
	}
}

// RenderEpollComparison outputs a comparison table between Select, Poll, and Epoll
func RenderEpollComparison(totalFDs, activeFDs int) string {
	var sb strings.Builder
	sb.WriteString(visualizer.SubHeader("I/O Multiplexing Scaling: Select/Poll vs Epoll"))

	tbl := visualizer.NewTable("I/O Mechanism", "Complexity", "Kernel Structure", "Scans on Event", "Active FD Retrieval", "C10K Scalability")
	tbl.SetAlignment("left", "center", "left", "center", "center", "center")

	tbl.AddRow("select()", "O(N)", "Bitmask (FD_SET)", fmt.Sprintf("%d FDs", totalFDs), "O(N) full scan", visualizer.Badge("POOR", visualizer.BgRed, visualizer.FgHiWhite))
	tbl.AddRow("poll()", "O(N)", "Array of pollfd", fmt.Sprintf("%d FDs", totalFDs), "O(N) full scan", visualizer.Badge("POOR", visualizer.BgRed, visualizer.FgHiWhite))
	tbl.AddRow("epoll() / kqueue", "O(1)", "RB-Tree + Ready-List", "0 (O(1))", "O(1) Ready-List only", visualizer.Badge("EXCELLENT", visualizer.BgGreen, visualizer.FgHiWhite))

	sb.WriteString(tbl.Render())

	deepDive := []string{
		"HOW EPOLL ACHIEVES O(1) PERFORMANCE UNDER THE HOOD:",
		"1. In-Kernel Interest List (Red-Black Tree):",
		"   - File descriptors are registered once with epoll_ctl(). No copying large arrays every loop!",
		"2. Device Driver Callback Integration:",
		"   - When a network packet arrives on the NIC, the hardware interrupt invokes ep_poll_callback().",
		"   - The kernel adds ONLY the activated socket to the epoll Ready List (Doubly-Linked List).",
		"3. epoll_wait() returns ONLY the Ready List:",
		"   - If 100,000 connections are idle and 2 receive data, epoll_wait returns in O(1) with 2 events!",
		"   - select() and poll() would wastefully iterate through all 100,000 descriptors.",
	}
	sb.WriteString("\n" + visualizer.Box("Epoll Architecture Deep Dive", deepDive))

	return sb.String()
}
