package kernel

import (
	"testing"
)

func TestProcessLifecycleForkExecWaitExit(t *testing.T) {
	pm := NewProcessManager()

	// 1. Fork init -> creates child PID 2
	child, err := pm.Fork(1)
	if err != nil {
		t.Fatalf("Fork failed: %v", err)
	}
	if child.PID != 2 || child.PPID != 1 {
		t.Errorf("Unexpected child PID/PPID: %d/%d", child.PID, child.PPID)
	}
	if child.Registers.RAX != 0 {
		t.Errorf("Expected child RAX=0, got %d", child.Registers.RAX)
	}

	// 2. Exec in child
	err = pm.Exec(2, "bash", []string{"bash", "-c", "echo hello"})
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}
	if child.Name != "bash" {
		t.Errorf("Expected name 'bash', got '%s'", child.Name)
	}

	// 3. Child exits with status 42 -> becomes ZOMBIE
	err = pm.Exit(2, 42)
	if err != nil {
		t.Fatalf("Exit failed: %v", err)
	}
	if child.State != ProcStateZombie {
		t.Errorf("Expected ZOMBIE state, got %v", child.State)
	}

	// 4. Parent reaps child via Wait
	reapedPID, exitCode, err := pm.Wait(1)
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}
	if reapedPID != 2 || exitCode != 42 {
		t.Errorf("Expected reaped PID 2 with exit code 42, got %d/%d", reapedPID, exitCode)
	}

	// Verify child PCB is destroyed
	if _, exists := pm.Processes[2]; exists {
		t.Errorf("Expected child PCB to be removed after wait/reap")
	}
}

func TestOrphanReparentingToInit(t *testing.T) {
	pm := NewProcessManager()

	// Init (PID 1) forks Parent (PID 2)
	parent, _ := pm.Fork(1)
	// Parent (PID 2) forks Child (PID 3)
	child, _ := pm.Fork(parent.PID)

	if child.PPID != 2 {
		t.Errorf("Expected child PPID 2, got %d", child.PPID)
	}

	// Parent (PID 2) exits BEFORE child (PID 3) -> child becomes ORPHAN
	pm.Exit(parent.PID, 0)

	// Child should now be adopted by init (PID 1)
	if child.PPID != 1 {
		t.Errorf("Expected orphan child to be reparented to init (PID 1), got %d", child.PPID)
	}
}
