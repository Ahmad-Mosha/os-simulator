package kernel

import (
	"testing"
)

func TestCPUModePrivilegeViolation(t *testing.T) {
	cpu := NewHardwareCPU()

	// 1. Normal user instruction should succeed in User Mode
	userInst := CPUInstruction{
		Opcode:       InstUserArithmetic,
		Description:  "ADD EAX, EBX",
		IsPrivileged: false,
	}
	if err := cpu.ExecuteInstruction(userInst); err != nil {
		t.Fatalf("Expected user instruction to succeed, got error: %v", err)
	}

	// 2. Privileged instruction should FAIL with privilege violation trap in User Mode
	privInst := CPUInstruction{
		Opcode:       InstPrivilegedHalt,
		Description:  "HLT (Stop CPU)",
		IsPrivileged: true,
	}
	err := cpu.ExecuteInstruction(privInst)
	if err == nil {
		t.Fatalf("Expected privilege violation error in user mode, but got nil")
	}

	// 3. Switch to Kernel Mode and try privileged instruction again -> should SUCCEED
	cpu.SwitchToKernelMode(0x80)
	if err := cpu.ExecuteInstruction(privInst); err != nil {
		t.Fatalf("Expected privileged instruction to succeed in Kernel Mode, got: %v", err)
	}

	cpu.ReturnToUserMode()
	if cpu.CurrentMode != ModeUser {
		t.Errorf("Expected return to User Mode, got %v", cpu.CurrentMode)
	}
}

func TestTrapTableDispatch(t *testing.T) {
	cpu := NewHardwareCPU()
	tt := NewTrapTable()

	// Dispatch a Syscall Trap (Vector 0x80)
	var args [6]uint64
	args[0] = 1 // stdout fd
	args[1] = 0x00405000 // buffer address
	args[2] = 12 // length

	ctx, err := tt.DispatchTrap(cpu, 0x80, 1, args)
	if err != nil {
		t.Fatalf("DispatchTrap failed: %v", err)
	}

	if ctx.Vector != 0x80 || ctx.Type != TrapSyscall {
		t.Errorf("Unexpected trap context: %+v", ctx)
	}

	idtRender := tt.RenderIDT()
	if len(idtRender) == 0 {
		t.Errorf("Expected non-empty IDT visualization")
	}
}
