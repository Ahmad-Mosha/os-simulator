package memory

import (
	"errors"
	"fmt"
	"os-simulator/pkg/visualizer"
	"strings"
)

var (
	ErrPageFault       = errors.New("page fault: page not present in physical memory")
	ErrProtectionFault = errors.New("protection fault: write attempt on read-only page")
	ErrInvalidPage     = errors.New("invalid page: virtual page not mapped")
)

// PageTableEntry (PTE) represents a hardware MMU page table entry
// OSTEP Chapter 18: Paging: Introduction
type PageTableEntry struct {
	Valid          bool   `json:"valid"`           // 1 if virtual page is mapped in address space
	Present        bool   `json:"present"`         // 1 if currently in physical RAM, 0 if swapped to disk
	ReadWrite      bool   `json:"read_write"`      // 1 = Read/Write, 0 = Read-Only
	UserSupervisor bool   `json:"user_supervisor"` // 1 = User mode accessible, 0 = Kernel only
	Accessed       bool   `json:"accessed"`        // Set by MMU on read/write (for Clock algorithm)
	Dirty          bool   `json:"dirty"`           // Set by MMU on write (must write back to swap)
	PFN            uint64 `json:"pfn"`             // Physical Frame Number
	SwapOffset     uint64 `json:"swap_offset"`     // Block index in swap file if !Present
}

// PageTable represents a process linear page table
type PageTable struct {
	PID       int
	PageSize  uint64 // Typically 4096 (4KB)
	Entries   []PageTableEntry
	NumPages  int
}

// NewPageTable initializes a single-level page table for a process
func NewPageTable(pid int, numPages int, pageSize uint64) *PageTable {
	if pageSize == 0 {
		pageSize = 4096 // 4 KB default page size
	}
	return &PageTable{
		PID:      pid,
		PageSize: pageSize,
		Entries:  make([]PageTableEntry, numPages),
		NumPages: numPages,
	}
}

// MapPage maps a Virtual Page Number (VPN) to a Physical Frame Number (PFN)
func (pt *PageTable) MapPage(vpn uint64, pfn uint64, writable bool) {
	if int(vpn) < len(pt.Entries) {
		pt.Entries[vpn] = PageTableEntry{
			Valid:          true,
			Present:        true,
			ReadWrite:      writable,
			UserSupervisor: true,
			PFN:            pfn,
		}
	}
}

// Translate performs MMU address translation from Virtual Address to Physical Address
func (pt *PageTable) Translate(vAddr uint64, isWrite bool) (pAddr uint64, pte *PageTableEntry, err error) {
	vpn := vAddr / pt.PageSize
	offset := vAddr % pt.PageSize

	if int(vpn) >= len(pt.Entries) || !pt.Entries[vpn].Valid {
		return 0, nil, fmt.Errorf("%w: VPN %d at vAddr 0x%X", ErrInvalidPage, vpn, vAddr)
	}

	entry := &pt.Entries[vpn]

	// Check Present bit (Page Fault)
	if !entry.Present {
		return 0, entry, fmt.Errorf("%w: VPN %d at vAddr 0x%X is swapped to disk (Swap Offset: %d)", ErrPageFault, vpn, vAddr, entry.SwapOffset)
	}

	// Check Permissions
	if isWrite && !entry.ReadWrite {
		return 0, entry, fmt.Errorf("%w: Cannot write to read-only page (VPN %d)", ErrProtectionFault, vpn)
	}

	// Update hardware tracking bits
	entry.Accessed = true
	if isWrite {
		entry.Dirty = true
	}

	// Compute physical address = (PFN * PageSize) + Offset
	pAddr = (entry.PFN * pt.PageSize) + offset
	return pAddr, entry, nil
}

// RenderPageTable outputs a formatted visual table of all valid page mappings
func (pt *PageTable) RenderPageTable() string {
	var sb strings.Builder
	sb.WriteString(visualizer.SubHeader(fmt.Sprintf("Process %d Linear Page Table (Page Size: %d bytes)", pt.PID, pt.PageSize)))

	tbl := visualizer.NewTable("VPN", "PFN", "Valid", "Present", "R/W", "Accessed", "Dirty", "Status")
	tbl.SetAlignment("center", "center", "center", "center", "center", "center", "center", "left")

	for vpn, pte := range pt.Entries {
		if !pte.Valid {
			continue
		}

		validStr := visualizer.Badge("1", visualizer.BgGreen, visualizer.FgHiWhite)
		presStr := visualizer.Badge("RAM", visualizer.BgGreen, visualizer.FgHiWhite)
		if !pte.Present {
			presStr = visualizer.Badge("DISK", visualizer.BgRed, visualizer.FgHiWhite)
		}

		rwStr := "RO"
		if pte.ReadWrite {
			rwStr = "RW"
		}

		accStr := "0"
		if pte.Accessed {
			accStr = "1"
		}

		dirtyStr := "0"
		if pte.Dirty {
			dirtyStr = "1"
		}

		status := "In Physical RAM"
		if !pte.Present {
			status = fmt.Sprintf("Swapped (SwapIdx:%d)", pte.SwapOffset)
		}

		tbl.AddRow(
			fmt.Sprintf("0x%04X (%d)", vpn, vpn),
			fmt.Sprintf("0x%04X (%d)", pte.PFN, pte.PFN),
			validStr,
			presStr,
			rwStr,
			accStr,
			dirtyStr,
			status,
		)
	}

	sb.WriteString(tbl.Render())
	return sb.String()
}
