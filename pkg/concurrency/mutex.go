package concurrency

import (
	"sync"
	"sync/atomic"
)

// Mutex implements a yielding/blocking mutex with wait queue (simulating Linux Futex)
// OSTEP Chapter 28: Locks: Two-Phase Locks / Futex (Fast Userspace Mutex)
// Phase 1: Spin briefly on atomic CAS.
// Phase 2: If still locked, park thread on OS wait queue to avoid wasting CPU.
type Mutex struct {
	state   int32       // 0 = Unlocked, 1 = Locked, 2 = Locked with waiters
	waiters []*waitNode // Wait queue for parked threads
	mu      sync.Mutex  // Internal queue guard
}

type waitNode struct {
	ready chan struct{}
}

func NewMutex() *Mutex {
	return &Mutex{
		state:   0,
		waiters: make([]*waitNode, 0),
	}
}

// Lock acquires the mutex, putting the calling goroutine to sleep if contested
func (m *Mutex) Lock() {
	// Fast path: try to acquire uncontended lock directly
	if atomic.CompareAndSwapInt32(&m.state, 0, 1) {
		return
	}

	// Slow path (Futex / Park on wait queue)
	node := &waitNode{ready: make(chan struct{})}

	m.mu.Lock()
	m.waiters = append(m.waiters, node)
	atomic.StoreInt32(&m.state, 2) // Flag that there are waiting threads
	m.mu.Unlock()

	// Park thread (blocks until awakened by Unlock)
	<-node.ready
}

// Unlock releases the mutex and awakens the next waiting thread
func (m *Mutex) Unlock() {
	m.mu.Lock()
	if len(m.waiters) > 0 {
		// Pop first waiting thread from FIFO queue
		next := m.waiters[0]
		m.waiters = m.waiters[1:]
		if len(m.waiters) == 0 {
			atomic.StoreInt32(&m.state, 1)
		}
		m.mu.Unlock()

		// Wake up the parked thread
		close(next.ready)
		return
	}

	atomic.StoreInt32(&m.state, 0)
	m.mu.Unlock()
}
