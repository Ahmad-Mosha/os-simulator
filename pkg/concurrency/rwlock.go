package concurrency

import (
	"sync"
)

// RWLock implements a Reader-Writer Lock with Writer-Preference to prevent writer starvation
// OSTEP Chapter 31: Semaphores: Reader-Writer Locks
type RWLock struct {
	readersCount int
	writerActive bool
	writerWaiters int
	mu           sync.Mutex
	readOK       *sync.Cond
	writeOK      *sync.Cond
}

func NewRWLock() *RWLock {
	rw := &RWLock{}
	rw.readOK = sync.NewCond(&rw.mu)
	rw.writeOK = sync.NewCond(&rw.mu)
	return rw
}

// RLock acquires shared read lock
func (rw *RWLock) RLock() {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	// Writer preference: Wait if a writer is active OR writers are queued
	for rw.writerActive || rw.writerWaiters > 0 {
		rw.readOK.Wait()
	}
	rw.readersCount++
}

// RUnlock releases shared read lock
func (rw *RWLock) RUnlock() {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	rw.readersCount--
	if rw.readersCount == 0 {
		rw.writeOK.Signal() // Signal waiting writer
	}
}

// Lock acquires exclusive write lock
func (rw *RWLock) Lock() {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	rw.writerWaiters++
	for rw.writerActive || rw.readersCount > 0 {
		rw.writeOK.Wait()
	}
	rw.writerWaiters--
	rw.writerActive = true
}

// Unlock releases exclusive write lock
func (rw *RWLock) Unlock() {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	rw.writerActive = false

	if rw.writerWaiters > 0 {
		rw.writeOK.Signal() // Prioritize next writer
	} else {
		rw.readOK.Broadcast() // Wake up all waiting readers
	}
}
