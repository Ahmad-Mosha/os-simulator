package kernel

import (
	"os-simulator/pkg/cpu"
	"testing"
)

func TestOSKernelStepAndDashboard(t *testing.T) {
	k := NewOSKernel(cpu.NewRRScheduler(2))

	// Step kernel ticks
	for i := 0; i < 5; i++ {
		msg := k.Step()
		if len(msg) == 0 {
			t.Fatalf("Expected step message")
		}
	}

	// Trigger a syscall
	var args [6]uint64
	ret, msg := k.TriggerSyscall(1, SysGetPID, args)
	if ret != 1 {
		t.Errorf("Expected SysGetPID to return 1, got %d", ret)
	}
	if len(msg) == 0 {
		t.Errorf("Expected non-empty syscall message")
	}

	dash := k.RenderDashboard()
	if len(dash) == 0 {
		t.Errorf("Expected non-empty dashboard")
	}
}
