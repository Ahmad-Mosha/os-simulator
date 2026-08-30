package memory

import (
	"testing"
)

func TestPageTableTranslation(t *testing.T) {
	pt := NewPageTable(1, 16, 4096)
	pt.MapPage(2, 5, true) // VPN 2 -> PFN 5

	vAddr := uint64(2*4096 + 100)
	pAddr, _, err := pt.Translate(vAddr, false)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	expected := uint64(5*4096 + 100)
	if pAddr != expected {
		t.Errorf("Expected physical address 0x%X, got 0x%X", expected, pAddr)
	}
}

func TestMultiLevelPageTable(t *testing.T) {
	mlpt := NewMultiLevelPageTable(1)
	vAddr := uint64(0x00401000) // maps to some PDE and PTE
	mlpt.Map(vAddr, 12, true)

	pAddr, _, err := mlpt.Translate(vAddr, false)
	if err != nil {
		t.Fatalf("Two-level translation failed: %v", err)
	}

	expected := uint64(12*4096 + (vAddr & 0xFFF))
	if pAddr != expected {
		t.Errorf("Expected physical addr 0x%X, got 0x%X", expected, pAddr)
	}
}

func TestTLBHitAndMiss(t *testing.T) {
	tlb := NewTLB(4)

	// Initial lookup should miss
	_, hit := tlb.Lookup(1, 10)
	if hit {
		t.Fatalf("Expected TLB miss on first access")
	}

	// Insert and lookup again -> should hit
	tlb.Insert(1, 10, 25, true)
	pfn, hit := tlb.Lookup(1, 10)
	if !hit || pfn != 25 {
		t.Fatalf("Expected TLB hit with PFN 25, got hit=%v, pfn=%d", hit, pfn)
	}

	if tlb.HitRate() != 50.0 {
		t.Errorf("Expected 50%% hit rate, got %f", tlb.HitRate())
	}
}

func TestMemoryManagerClockReplacement(t *testing.T) {
	mm := NewMemoryManager(3, PolicyClock, 4)
	pt := NewPageTable(1, 10, 4096)
	for i := 0; i < 10; i++ {
		pt.MapPage(uint64(i), 0, true)
		pt.Entries[i].Present = false // All swapped initially
	}

	// Access pages 0, 1, 2 (fills 3 frames)
	mm.AccessMemory(pt, 0*4096, false)
	mm.AccessMemory(pt, 1*4096, false)
	mm.AccessMemory(pt, 2*4096, false)

	if mm.PageFaults != 3 {
		t.Errorf("Expected 3 page faults, got %d", mm.PageFaults)
	}

	// Access page 3 -> causes clock eviction
	mm.AccessMemory(pt, 3*4096, false)

	if mm.PageEvictions != 1 {
		t.Errorf("Expected 1 page eviction, got %d", mm.PageEvictions)
	}
}
