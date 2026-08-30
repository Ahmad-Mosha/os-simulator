package kernel

import (
	"testing"
)

func TestSyscallDispatcher(t *testing.T) {
	sd := NewSyscallDispatcher()

	// 1. Test getpid syscall (RAX = SysGetPID)
	regs := &SyscallRegisters{
		RAX: uint64(SysGetPID),
	}
	ret, err := sd.Dispatch(42, regs)
	if err != nil {
		t.Fatalf("getpid syscall failed: %v", err)
	}
	if ret != 42 || regs.RAX != 42 {
		t.Errorf("Expected getpid to return 42, got %d", ret)
	}

	// 2. Test valid write syscall
	regsWrite := &SyscallRegisters{
		RAX: uint64(SysWrite),
		RDI: 1,          // stdout
		RSI: 0x00405000, // Valid user space address
		RDX: 64,         // bytes
	}
	retWrite, err := sd.Dispatch(42, regsWrite)
	if err != nil {
		t.Fatalf("write syscall failed: %v", err)
	}
	if retWrite != 64 {
		t.Errorf("Expected write to return 64 bytes, got %d", retWrite)
	}

	// 3. Test invalid user pointer in write syscall (pointing into kernel space 0xC0001000)
	regsBadWrite := &SyscallRegisters{
		RAX: uint64(SysWrite),
		RDI: 1,
		RSI: 0xC0001000, // Bad pointer into kernel space!
		RDX: 64,
	}
	_, err = sd.Dispatch(42, regsBadWrite)
	if err == nil {
		t.Fatalf("Expected write with kernel pointer to fail, but succeeded")
	}

	tableRender := sd.RenderSyscallTable()
	if len(tableRender) == 0 {
		t.Errorf("Expected non-empty rendered syscall table")
	}
}
