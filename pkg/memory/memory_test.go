package memory

import (
	"testing"
)

func TestAddressSpaceAndSbrk(t *testing.T) {
	as := NewAddressSpace(1, 64*1024, 0x100000)

	// Test translation
	pAddr, err := as.TranslateBaseAndBounds(0x100)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}
	expected := uint64(0x100000 + 0x100)
	if pAddr != expected {
		t.Errorf("Expected physical addr 0x%X, got 0x%X", expected, pAddr)
	}

	// Test Sbrk
	oldBreak, err := as.Sbrk(1024)
	if err != nil {
		t.Fatalf("Sbrk failed: %v", err)
	}
	if oldBreak != as.HeapStart {
		t.Errorf("Expected old break 0x%X, got 0x%X", as.HeapStart, oldBreak)
	}
}

func TestCallStackPushPop(t *testing.T) {
	cs := NewCallStack(0xFFFF, 16*1024)

	locals := map[string]interface{}{"x": 42, "msg": "hello"}
	frame, err := cs.PushFrame("main", 0x00400000, 64, locals)
	if err != nil {
		t.Fatalf("PushFrame failed: %v", err)
	}
	if frame.FunctionName != "main" {
		t.Errorf("Expected function 'main', got '%s'", frame.FunctionName)
	}

	locals2 := map[string]interface{}{"result": 100}
	_, err = cs.PushFrame("calculate", 0x00401020, 32, locals2)
	if err != nil {
		t.Fatalf("PushFrame calculate failed: %v", err)
	}

	if len(cs.Frames) != 2 {
		t.Fatalf("Expected 2 frames, got %d", len(cs.Frames))
	}

	popped, err := cs.PopFrame()
	if err != nil {
		t.Fatalf("PopFrame failed: %v", err)
	}
	if popped.FunctionName != "calculate" {
		t.Errorf("Expected popped function 'calculate', got '%s'", popped.FunctionName)
	}
}

func TestFreeListAllocatorCoalescing(t *testing.T) {
	fla := NewFreeListAllocator(0x1000, 4096)

	// Allocate 3 blocks
	a1, err := fla.Malloc(512, false)
	if err != nil {
		t.Fatalf("Malloc a1 failed: %v", err)
	}
	a2, err := fla.Malloc(512, false)
	if err != nil {
		t.Fatalf("Malloc a2 failed: %v", err)
	}
	a3, err := fla.Malloc(512, false)
	if err != nil {
		t.Fatalf("Malloc a3 failed: %v", err)
	}

	// Free middle block then adjacent block -> test coalescing
	if err := fla.Free(a2); err != nil {
		t.Fatalf("Free a2 failed: %v", err)
	}
	if err := fla.Free(a1); err != nil {
		t.Fatalf("Free a1 failed: %v", err)
	}

	// Now allocate 1024 bytes -> should fit in coalesced space
	aMerged, err := fla.Malloc(1024, false)
	if err != nil {
		t.Fatalf("Malloc aMerged failed after coalescing: %v", err)
	}
	if aMerged != a1 {
		t.Errorf("Expected merged block at 0x%X, got 0x%X", a1, aMerged)
	}

	_ = a3
}

func TestBuddyAllocator(t *testing.T) {
	ba := NewBuddyAllocator(0x0, 4, 10) // 16 bytes min to 1024 bytes max

	addr1, err := ba.Allocate(30) // should get 32-byte block (order 5)
	if err != nil {
		t.Fatalf("Buddy allocate failed: %v", err)
	}
	if addr1 != 0 {
		t.Errorf("Expected first allocation at 0, got %d", addr1)
	}
}
