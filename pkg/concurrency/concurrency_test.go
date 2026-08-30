package concurrency

import (
	"sync"
	"testing"
)

func TestDataRaceDemo(t *testing.T) {
	res := RunDataRaceDemo(10, 1000)
	if res.SafeActualValue != res.ExpectedValue {
		t.Errorf("Expected safe counter %d, got %d", res.ExpectedValue, res.SafeActualValue)
	}

	explanation := RenderRaceExplanation(res)
	if len(explanation) == 0 {
		t.Errorf("Expected non-empty race explanation")
	}
}

func TestHardwareAtomics(t *testing.T) {
	ha := &HardwareAtomics{}
	var val int32 = 10

	if !ha.CompareAndSwap(&val, 10, 20) {
		t.Fatalf("CAS 10->20 failed")
	}
	if ha.Load(&val) != 20 {
		t.Errorf("Expected val 20, got %d", ha.Load(&val))
	}

	old := ha.FetchAndAdd(&val, 5)
	if old != 20 || ha.Load(&val) != 25 {
		t.Errorf("FetchAndAdd failed: old=%d, new=%d", old, ha.Load(&val))
	}
}

func TestSpinlockAndMutexMutualExclusion(t *testing.T) {
	// 1. Test Spinlock
	spin := NewSpinlock()
	var spinCounter int
	var wg sync.WaitGroup
	numWorkers := 10
	iters := 500

	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				spin.Lock()
				spinCounter++
				spin.Unlock()
			}
		}()
	}
	wg.Wait()

	if spinCounter != numWorkers*iters {
		t.Errorf("Spinlock counter mismatch: expected %d, got %d", numWorkers*iters, spinCounter)
	}

	// 2. Test Mutex
	mut := NewMutex()
	var mutCounter int
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				mut.Lock()
				mutCounter++
				mut.Unlock()
			}
		}()
	}
	wg.Wait()

	if mutCounter != numWorkers*iters {
		t.Errorf("Mutex counter mismatch: expected %d, got %d", numWorkers*iters, mutCounter)
	}
}

func TestBoundedBufferProducerConsumer(t *testing.T) {
	buf := NewBoundedBuffer(5)
	totalItems := 50
	var wg sync.WaitGroup

	// Producer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < totalItems; i++ {
			buf.Put(i)
		}
	}()

	// Consumer
	consumed := make([]int, 0, totalItems)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < totalItems; i++ {
			consumed = append(consumed, buf.Get())
		}
	}()

	wg.Wait()

	if len(consumed) != totalItems {
		t.Fatalf("Expected %d consumed items, got %d", totalItems, len(consumed))
	}
	for i := 0; i < totalItems; i++ {
		if consumed[i] != i {
			t.Errorf("Item %d mismatch: got %d", i, consumed[i])
		}
	}
}

func TestSemaphoreAndRWLock(t *testing.T) {
	// Test Semaphore
	sem := NewSemaphore(2)
	sem.Wait()
	sem.Wait()
	if sem.Value() != 0 {
		t.Errorf("Expected semaphore value 0, got %d", sem.Value())
	}
	sem.Signal()
	if sem.Value() != 1 {
		t.Errorf("Expected semaphore value 1, got %d", sem.Value())
	}

	// Test RWLock
	rw := NewRWLock()
	rw.RLock()
	rw.RLock()
	rw.RUnlock()
	rw.RUnlock()

	rw.Lock()
	rw.Unlock()
}
