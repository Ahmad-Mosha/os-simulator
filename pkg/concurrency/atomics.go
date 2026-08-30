package concurrency

import (
	"sync/atomic"
)

// HardwareAtomics simulates low-level CPU atomic memory bus operations (LOCK prefix on x86)
// OSTEP Chapter 28: Locks: Hardware-Assisted Locks
type HardwareAtomics struct{}

// CompareAndSwap wraps atomic CAS (LOCK CMPXCHG)
// If *addr == old, sets *addr = new and returns true; otherwise returns false without modifying *addr.
func (ha *HardwareAtomics) CompareAndSwap(addr *int32, old, new int32) bool {
	return atomic.CompareAndSwapInt32(addr, old, new)
}

// TestAndSet wraps atomic test-and-set (XCHG instruction)
// Atomically sets *addr = new and returns the previous value stored at *addr.
func (ha *HardwareAtomics) TestAndSet(addr *int32, new int32) int32 {
	return atomic.SwapInt32(addr, new)
}

// FetchAndAdd wraps atomic fetch-and-add (LOCK XADD)
// Atomically adds delta to *addr and returns the previous value.
func (ha *HardwareAtomics) FetchAndAdd(addr *int32, delta int32) int32 {
	return atomic.AddInt32(addr, delta) - delta
}

// Load atomically reads value from memory
func (ha *HardwareAtomics) Load(addr *int32) int32 {
	return atomic.LoadInt32(addr)
}

// Store atomically writes value to memory
func (ha *HardwareAtomics) Store(addr *int32, val int32) {
	atomic.StoreInt32(addr, val)
}
