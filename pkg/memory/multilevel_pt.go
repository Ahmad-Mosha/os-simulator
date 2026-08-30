package memory

import (
	"fmt"
	"os-simulator/pkg/visualizer"
	"strings"
)

// PageDirectoryEntry represents an entry in the top-level Page Directory
type PageDirectoryEntry struct {
	Valid     bool
	PageTable *PageTable
	PFN       uint64
}

// MultiLevelPageTable implements a Two-Level Hierarchical Page Table (OSTEP Chapter 20)
// Resolves linear page table memory bloat by allocating second-level page tables on-demand.
// Address structure for 32-bit:
// [ 10 bits: PDE Index ] [ 10 bits: PTE Index ] [ 12 bits: Page Offset (4KB) ]
type MultiLevelPageTable struct {
	PID           int
	PageSize      uint64 // 4096
	DirectorySize int    // 1024 entries (10 bits)
	TableSize     int    // 1024 entries per table (10 bits)
	Directory     []PageDirectoryEntry
	AllocatedPTs  int
}

func NewMultiLevelPageTable(pid int) *MultiLevelPageTable {
	return &MultiLevelPageTable{
		PID:           pid,
		PageSize:      4096,
		DirectorySize: 1024,
		TableSize:     1024,
		Directory:     make([]PageDirectoryEntry, 1024),
	}
}

// Map maps a virtual address to a physical frame, allocating the second-level table only if needed
func (mlpt *MultiLevelPageTable) Map(vAddr uint64, pfn uint64, writable bool) {
	pdeIdx := (vAddr >> 22) & 0x3FF
	pteIdx := (vAddr >> 12) & 0x3FF

	if !mlpt.Directory[pdeIdx].Valid {
		mlpt.Directory[pdeIdx] = PageDirectoryEntry{
			Valid:     true,
			PageTable: NewPageTable(mlpt.PID, mlpt.TableSize, mlpt.PageSize),
			PFN:       pdeIdx + 100,
		}
		mlpt.AllocatedPTs++
	}

	mlpt.Directory[pdeIdx].PageTable.MapPage(pteIdx, pfn, writable)
}

// Translate performs two-level address translation: PDE -> PTE -> Physical Address
func (mlpt *MultiLevelPageTable) Translate(vAddr uint64, isWrite bool) (uint64, *PageTableEntry, error) {
	pdeIdx := (vAddr >> 22) & 0x3FF
	pteIdx := (vAddr >> 12) & 0x3FF
	offset := vAddr & 0xFFF

	if !mlpt.Directory[pdeIdx].Valid {
		return 0, nil, fmt.Errorf("%w: Page Directory Entry %d not mapped for vAddr 0x%08X", ErrInvalidPage, pdeIdx, vAddr)
	}

	pt := mlpt.Directory[pdeIdx].PageTable
	if int(pteIdx) >= len(pt.Entries) || !pt.Entries[pteIdx].Valid {
		return 0, nil, fmt.Errorf("%w: Page Table Entry %d not valid in PDE %d", ErrInvalidPage, pteIdx, pdeIdx)
	}

	entry := &pt.Entries[pteIdx]
	if !entry.Present {
		return 0, entry, fmt.Errorf("%w: VPN [PDE:%d, PTE:%d] swapped to disk", ErrPageFault, pdeIdx, pteIdx)
	}

	if isWrite && !entry.ReadWrite {
		return 0, entry, fmt.Errorf("%w: Read-only page violation at vAddr 0x%08X", ErrProtectionFault, vAddr)
	}

	entry.Accessed = true
	if isWrite {
		entry.Dirty = true
	}

	pAddr := (entry.PFN * mlpt.PageSize) + offset
	return pAddr, entry, nil
}

// RenderTwoLevelStatus compares memory consumed by linear vs two-level page tables
func (mlpt *MultiLevelPageTable) RenderTwoLevelStatus() string {
	var sb strings.Builder
	sb.WriteString(visualizer.SubHeader("Two-Level Hierarchical Page Table (OSTEP Chapter 20)"))

	linearPTBytes := 1024 * 1024 * 4 // 4MB for a 32-bit linear page table (1M entries * 4 bytes)
	multiLevelPTBytes := (1024 * 4) + (mlpt.AllocatedPTs * 1024 * 4) // Directory (4KB) + active page tables (4KB each)

	savingsPercent := 100.0 - (float64(multiLevelPTBytes) / float64(linearPTBytes) * 100.0)

	lines := []string{
		fmt.Sprintf("Active Page Directory Entries: %d / %d", mlpt.AllocatedPTs, mlpt.DirectorySize),
		fmt.Sprintf("Allocated Level-2 Page Tables: %d", mlpt.AllocatedPTs),
		fmt.Sprintf("Memory Used by Linear PT:      %d KB (4 MB flat array, mostly empty zeros!)", linearPTBytes/1024),
		fmt.Sprintf("Memory Used by Multi-Level PT: %d KB (Directory + only allocated sub-tables)", multiLevelPTBytes/1024),
		fmt.Sprintf("Memory Savings:                %.2f%% space reduction!", savingsPercent),
		"",
		"WHY MULTI-LEVEL PAGING IS CRITICAL:",
		"• Most processes only use a tiny fraction of their virtual address space (Code + Small Heap + Stack).",
		"• Flat linear page tables force the OS to allocate page tables for all 4GB (or 256TB in 64-bit).",
		"• Multi-level paging allocates sub-tables ON-DEMAND, creating huge memory savings for sparse spaces.",
	}

	sb.WriteString(visualizer.Box("Multi-Level Page Table Memory Efficiency", lines))
	return sb.String()
}
