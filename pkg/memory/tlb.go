package memory

import (
	"fmt"
	"os-simulator/pkg/visualizer"
	"strings"
	"sync"
)

// TLBEntry represents a single hardware MMU cache line in the TLB
type TLBEntry struct {
	VPN          uint64
	PFN          uint64
	ASID         int  // Address Space Identifier (Process ID) to avoid full TLB flush on context switch
	Valid        bool
	ReadWrite    bool
	LastAccess   int64 // For LRU replacement
}

// TLB (Translation Lookaside Buffer) simulates the CPU MMU's fast translation hardware cache
// OSTEP Chapter 19: Paging: Faster Translations (TLBs)
type TLB struct {
	Capacity   int
	Entries    []TLBEntry
	Hits       int
	Misses     int
	AccessTick int64
	mu         sync.Mutex
}

func NewTLB(capacity int) *TLB {
	if capacity <= 0 {
		capacity = 16 // Typical hardware L1 TLB holds 16-64 entries
	}
	return &TLB{
		Capacity: capacity,
		Entries:  make([]TLBEntry, capacity),
	}
}

// Lookup searches TLB for a valid mapping (1 CPU cycle cost)
func (tlb *TLB) Lookup(pid int, vpn uint64) (pfn uint64, found bool) {
	tlb.mu.Lock()
	defer tlb.mu.Unlock()

	tlb.AccessTick++

	for i := range tlb.Entries {
		e := &tlb.Entries[i]
		if e.Valid && e.ASID == pid && e.VPN == vpn {
			e.LastAccess = tlb.AccessTick
			tlb.Hits++
			return e.PFN, true
		}
	}

	tlb.Misses++
	return 0, false
}

// Insert caches a translation into TLB using LRU eviction if full
func (tlb *TLB) Insert(pid int, vpn, pfn uint64, writable bool) {
	tlb.mu.Lock()
	defer tlb.mu.Unlock()

	tlb.AccessTick++

	// 1. Try to find empty slot or replace existing entry
	lruIdx := 0
	oldestAccess := tlb.AccessTick + 1

	for i := range tlb.Entries {
		e := &tlb.Entries[i]
		if !e.Valid {
			// Found empty slot
			tlb.Entries[i] = TLBEntry{
				VPN:        vpn,
				PFN:        pfn,
				ASID:       pid,
				Valid:      true,
				ReadWrite:  writable,
				LastAccess: tlb.AccessTick,
			}
			return
		}
		if e.Valid && e.ASID == pid && e.VPN == vpn {
			// Update existing
			e.PFN = pfn
			e.ReadWrite = writable
			e.LastAccess = tlb.AccessTick
			return
		}
		if e.LastAccess < oldestAccess {
			oldestAccess = e.LastAccess
			lruIdx = i
		}
	}

	// 2. Evict LRU entry
	tlb.Entries[lruIdx] = TLBEntry{
		VPN:        vpn,
		PFN:        pfn,
		ASID:       pid,
		Valid:      true,
		ReadWrite:  writable,
		LastAccess: tlb.AccessTick,
	}
}

// Flush clears all TLB entries (performed on process context switch when hardware lacks ASID)
func (tlb *TLB) Flush() {
	tlb.mu.Lock()
	defer tlb.mu.Unlock()

	for i := range tlb.Entries {
		tlb.Entries[i].Valid = false
	}
}

// HitRate returns percentage of accesses served by the TLB
func (tlb *TLB) HitRate() float64 {
	total := tlb.Hits + tlb.Misses
	if total == 0 {
		return 0.0
	}
	return (float64(tlb.Hits) / float64(total)) * 100.0
}

// RenderTLBStatus outputs formatted visual table of current TLB entries and hit rates
func (tlb *TLB) RenderTLBStatus() string {
	var sb strings.Builder
	sb.WriteString(visualizer.SubHeader(fmt.Sprintf("MMU Hardware TLB Cache (Capacity: %d entries)", tlb.Capacity)))

	tbl := visualizer.NewTable("Slot", "ASID (PID)", "VPN", "PFN", "R/W", "Valid", "Last Access")
	tbl.SetAlignment("center", "center", "center", "center", "center", "center", "right")

	for i, e := range tlb.Entries {
		if !e.Valid {
			tbl.AddRow(fmt.Sprintf("#%d", i), "-", "-", "-", "-", visualizer.Badge("INVALID", visualizer.BgBlack, visualizer.FgHiBlack), "-")
			continue
		}

		validBadge := visualizer.Badge("VALID", visualizer.BgGreen, visualizer.FgHiWhite)
		rwStr := "RO"
		if e.ReadWrite {
			rwStr = "RW"
		}

		tbl.AddRow(
			fmt.Sprintf("#%d", i),
			fmt.Sprintf("%d", e.ASID),
			fmt.Sprintf("0x%04X", e.VPN),
			fmt.Sprintf("0x%04X", e.PFN),
			rwStr,
			validBadge,
			fmt.Sprintf("%d", e.LastAccess),
		)
	}

	sb.WriteString(tbl.Render())

	totalAccesses := tlb.Hits + tlb.Misses
	avgCycles := 1.0 + (float64(tlb.Misses)/float64(maxInt(1, totalAccesses)))*100.0 // 1 cycle hit, 100 cycles page table walk miss

	stats := []string{
		fmt.Sprintf("Total Lookups:     %d", totalAccesses),
		fmt.Sprintf("TLB Hits:          %s%d%s (Cost: 1 cycle per hit)", visualizer.FgHiGreen, tlb.Hits, visualizer.Reset),
		fmt.Sprintf("TLB Misses:        %s%d%s (Cost: ~100-200 cycles to walk page tables in RAM)", visualizer.FgHiRed, tlb.Misses, visualizer.Reset),
		fmt.Sprintf("TLB Hit Rate:      %s%.2f%%%s", visualizer.Bold+visualizer.FgHiCyan, tlb.HitRate(), visualizer.Reset),
		fmt.Sprintf("Effective Access:  %.1f CPU cycles (vs 100 cycles without TLB)", avgCycles),
		"",
		"WHY TLB IS CRITICAL FOR OS PERFORMANCE:",
		"• Without TLB: EVERY single memory load/store requires 2-4 memory accesses (page table walks) -> 300% slowdown!",
		"• With TLB: >99% hit rate -> nearly all memory accesses translate at hardware speed (1 cycle)!",
	}

	sb.WriteString("\n" + visualizer.Box("TLB Hardware Performance Metrics", stats))
	return sb.String()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
