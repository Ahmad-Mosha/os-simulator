package concurrency

import (
	"runtime"
	"sync/atomic"
)

// Spinlock implements a simple busy-waiting spinlock using atomic TestAndSet
// OSTEP Chapter 28: Locks: Spin Locks
type Spinlock struct {
	state int32 // 0 = Unlocked, 1 = Locked
}

func NewSpinlock() *Spinlock {
	return &Spinlock{state: 0}
}

// Lock spins in a tight loop until acquiring the lock
func (s *Spinlock) Lock() {
	for !atomic.CompareAndSwapInt32(&s.state, 0, 1) {
		// In real CPUs, a PAUSE instruction or processor yield reduces power consumption
		runtime.Gosched() // Yield CPU slice to avoid burning 100% core cycles in user-space
	}
}

// TryLock attempts to acquire the lock once without spinning
func (s *Spinlock) TryLock() bool {
	return atomic.CompareAndSwapInt32(&s.state, 0, 1)
}

// Unlock releases the spinlock
func (s *Spinlock) Unlock() {
	atomic.StoreInt32(&s.state, 0)
}
