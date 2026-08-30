package concurrency

import (
	"sync"
)

// Semaphore implements Dijkstra's Counting Semaphore
// OSTEP Chapter 31: Semaphores
// Semaphore has an integer value manipulated by two atomic operations:
// P (Wait / Decrement): if value <= 0, caller blocks.
// V (Signal / Post / Increment): increments value; unblocks one waiting thread.
type Semaphore struct {
	value   int
	waiters []chan struct{}
	mu      sync.Mutex
}

func NewSemaphore(initialValue int) *Semaphore {
	return &Semaphore{
		value:   initialValue,
		waiters: make([]chan struct{}, 0),
	}
}

// Wait (P operation) decrements the semaphore; blocks if value is <= 0
func (s *Semaphore) Wait() {
	s.mu.Lock()
	s.value--
	if s.value < 0 {
		ch := make(chan struct{})
		s.waiters = append(s.waiters, ch)
		s.mu.Unlock()
		<-ch // Block caller
		return
	}
	s.mu.Unlock()
}

// Signal (V operation) increments the semaphore; unblocks a waiting thread if any
func (s *Semaphore) Signal() {
	s.mu.Lock()
	s.value++
	if len(s.waiters) > 0 {
		ch := s.waiters[0]
		s.waiters = s.waiters[1:]
		s.mu.Unlock()
		close(ch) // Wake up blocked thread
		return
	}
	s.mu.Unlock()
}

// Value returns current semaphore count
func (s *Semaphore) Value() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.value
}
