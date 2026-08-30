package cpu

import (
	"testing"
)

func TestGMPRuntimeWorkStealingAndExecution(t *testing.T) {
	gomaxprocs := 2
	rt := NewGMPRuntime(gomaxprocs)

	// Spawn multiple goroutines
	for i := 0; i < 6; i++ {
		rt.SpawnGoroutine("worker", 3, false)
	}

	// Also spawn a syscall goroutine to test P handoff
	rt.SpawnGoroutine("io_syscall", 5, true)

	// Step through runtime execution
	for tick := 0; tick < 30; tick++ {
		rt.Step(tick)
	}

	if len(rt.CompletedGs) == 0 {
		t.Fatalf("Expected completed goroutines, got 0")
	}

	status := rt.RenderStatus()
	if len(status) == 0 {
		t.Errorf("Expected non-empty rendered status")
	}
}
