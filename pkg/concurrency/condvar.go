package concurrency

import (
	"sync"
)

// CondVar implements Condition Variables with Mesa semantics
// OSTEP Chapter 30: Condition Variables
// Mesa semantics rule: Signaling wakes up a thread, but the woken thread MUST re-check the condition in a 'while' loop!
// Because between the signal and when the woken thread actually runs, another thread might have changed the state.
type CondVar struct {
	mu      *Mutex
	waiters []chan struct{}
	queueMu sync.Mutex
}

func NewCondVar(mu *Mutex) *CondVar {
	return &CondVar{
		mu:      mu,
		waiters: make([]chan struct{}, 0),
	}
}

// Wait atomically releases the associated lock and blocks until signaled, then re-acquires the lock
func (cv *CondVar) Wait() {
	ch := make(chan struct{})

	cv.queueMu.Lock()
	cv.waiters = append(cv.waiters, ch)
	cv.queueMu.Unlock()

	// 1. Release associated lock
	cv.mu.Unlock()

	// 2. Wait to be signaled
	<-ch

	// 3. Re-acquire lock before returning to caller
	cv.mu.Lock()
}

// Signal wakes up ONE waiting thread
func (cv *CondVar) Signal() {
	cv.queueMu.Lock()
	defer cv.queueMu.Unlock()

	if len(cv.waiters) > 0 {
		ch := cv.waiters[0]
		cv.waiters = cv.waiters[1:]
		close(ch)
	}
}

// Broadcast wakes up ALL waiting threads
func (cv *CondVar) Broadcast() {
	cv.queueMu.Lock()
	defer cv.queueMu.Unlock()

	for _, ch := range cv.waiters {
		close(ch)
	}
	cv.waiters = make([]chan struct{}, 0)
}

// BoundedBuffer implements the classic Producer-Consumer Problem using Condition Variables
type BoundedBuffer struct {
	capacity int
	buffer   []int
	mu       *Mutex
	notFull  *CondVar
	notEmpty *CondVar
}

func NewBoundedBuffer(capacity int) *BoundedBuffer {
	mu := NewMutex()
	return &BoundedBuffer{
		capacity: capacity,
		buffer:   make([]int, 0, capacity),
		mu:       mu,
		notFull:  NewCondVar(mu),
		notEmpty: NewCondVar(mu),
	}
}

// Put inserts an item into the buffer (blocks if buffer is full)
func (bb *BoundedBuffer) Put(item int) {
	bb.mu.Lock()
	defer bb.mu.Unlock()

	// Mesa Semantics: MUST USE WHILE / FOR LOOP (not 'if')!
	for len(bb.buffer) == bb.capacity {
		bb.notFull.Wait()
	}

	bb.buffer = append(bb.buffer, item)
	bb.notEmpty.Signal() // Signal consumer that buffer is not empty
}

// Get extracts an item from the buffer (blocks if buffer is empty)
func (bb *BoundedBuffer) Get() int {
	bb.mu.Lock()
	defer bb.mu.Unlock()

	// Mesa Semantics: MUST USE WHILE / FOR LOOP!
	for len(bb.buffer) == 0 {
		bb.notEmpty.Wait()
	}

	item := bb.buffer[0]
	bb.buffer = bb.buffer[1:]
	bb.notFull.Signal() // Signal producer that buffer is not full

	return item
}
