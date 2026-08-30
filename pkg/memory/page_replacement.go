package memory

import (
	"fmt"
	"os-simulator/pkg/visualizer"
	"strings"
)

// ReplacementPolicy enum for page eviction strategies
type ReplacementPolicy string

const (
	PolicyFIFO  ReplacementPolicy = "FIFO"
	PolicyLRU   ReplacementPolicy = "LRU"
	PolicyClock ReplacementPolicy = "CLOCK"
)

// FrameInfo holds metadata for a physical memory frame
type FrameInfo struct {
	PFN          uint64
	PID          int
	VPN          uint64
	IsOccupied   bool
	LoadedTick   int64
	LastUsedTick int64
	UseBit       bool // For Clock (Second-Chance) algorithm
	IsDirty      bool
}

// MemoryManager coordinates Virtual Memory, Physical Frames, Page Faults, and Swap Space
// OSTEP Chapter 21: Beyond Physical Memory: Mechanisms & Chapter 22: Policies
type MemoryManager struct {
	NumFrames       int
	Frames          []FrameInfo
	ClockHand       int
	CurrentTick     int64
	Policy          ReplacementPolicy
	PageFaults      int
	PageEvictions   int
	SwapWrites      int // Writes back to disk when dirty page evicted
	SwapReads       int // Reads from disk when page faulted in
	TLB             *TLB
	History         []string
}

func NewMemoryManager(numFrames int, policy ReplacementPolicy, tlbCapacity int) *MemoryManager {
	if numFrames <= 0 {
		numFrames = 4 // Small frame count for easy visual tracing
	}
	frames := make([]FrameInfo, numFrames)
	for i := range frames {
		frames[i] = FrameInfo{PFN: uint64(i)}
	}

	return &MemoryManager{
		NumFrames: numFrames,
		Frames:    frames,
		Policy:    policy,
		TLB:       NewTLB(tlbCapacity),
		History:   make([]string, 0),
	}
}

// AccessMemory simulates an instruction or data access to a virtual address
func (mm *MemoryManager) AccessMemory(pt *PageTable, vAddr uint64, isWrite bool) (uint64, error) {
	mm.CurrentTick++
	vpn := vAddr / pt.PageSize
	offset := vAddr % pt.PageSize

	// 1. Try Hardware TLB Lookup first
	if pfn, hit := mm.TLB.Lookup(pt.PID, vpn); hit {
		pAddr := (pfn * pt.PageSize) + offset
		// Update frame access metadata
		frame := &mm.Frames[pfn]
		frame.LastUsedTick = mm.CurrentTick
		frame.UseBit = true
		if isWrite {
			frame.IsDirty = true
		}
		return pAddr, nil
	}

	// 2. TLB Miss -> Check Page Table
	if int(vpn) >= len(pt.Entries) || !pt.Entries[vpn].Valid {
		return 0, fmt.Errorf("%w: VPN %d not mapped", ErrInvalidPage, vpn)
	}

	pte := &pt.Entries[vpn]

	// 3. Check if Page is in Physical RAM (Present bit)
	if !pte.Present {
		// PAGE FAULT OCCURS!
		mm.PageFaults++
		mm.SwapReads++

		// Allocate a physical frame (may cause page eviction)
		allocatedPFN, evictedVPN, wasEvicted := mm.allocateFrame(pt.PID, vpn)
		
		event := fmt.Sprintf("[Tick %d] PAGE FAULT: VPN %d loaded into PFN %d", mm.CurrentTick, vpn, allocatedPFN)
		if wasEvicted {
			event += fmt.Sprintf(" (Evicted VPN %d via %s)", evictedVPN, mm.Policy)
		}
		mm.History = append(mm.History, event)

		// Update PTE
		pte.Present = true
		pte.PFN = allocatedPFN
		pte.Accessed = true
	}

	// 4. Update TLB with the newly resolved translation
	mm.TLB.Insert(pt.PID, vpn, pte.PFN, pte.ReadWrite)

	frame := &mm.Frames[pte.PFN]
	frame.LastUsedTick = mm.CurrentTick
	frame.UseBit = true
	if isWrite {
		frame.IsDirty = true
		pte.Dirty = true
	}

	pAddr := (pte.PFN * pt.PageSize) + offset
	return pAddr, nil
}

func (mm *MemoryManager) allocateFrame(pid int, vpn uint64) (pfn uint64, evictedVPN uint64, wasEvicted bool) {
	// Look for free frame first
	for i := range mm.Frames {
		if !mm.Frames[i].IsOccupied {
			mm.Frames[i] = FrameInfo{
				PFN:          uint64(i),
				PID:          pid,
				VPN:          vpn,
				IsOccupied:   true,
				LoadedTick:   mm.CurrentTick,
				LastUsedTick: mm.CurrentTick,
				UseBit:       true,
			}
			return uint64(i), 0, false
		}
	}

	// All frames occupied -> Run Page Replacement Policy
	mm.PageEvictions++
	victimIdx := 0

	switch mm.Policy {
	case PolicyFIFO:
		// Evict oldest loaded frame
		oldest := mm.Frames[0].LoadedTick
		for i := 1; i < mm.NumFrames; i++ {
			if mm.Frames[i].LoadedTick < oldest {
				oldest = mm.Frames[i].LoadedTick
				victimIdx = i
			}
		}

	case PolicyLRU:
		// Evict least recently accessed frame
		oldestAccess := mm.Frames[0].LastUsedTick
		for i := 1; i < mm.NumFrames; i++ {
			if mm.Frames[i].LastUsedTick < oldestAccess {
				oldestAccess = mm.Frames[i].LastUsedTick
				victimIdx = i
			}
		}

	case PolicyClock:
		// Clock / Second-Chance Algorithm (OSTEP Chapter 22)
		for {
			f := &mm.Frames[mm.ClockHand]
			if !f.UseBit {
				// Use bit is 0: Found victim!
				victimIdx = mm.ClockHand
				mm.ClockHand = (mm.ClockHand + 1) % mm.NumFrames
				break
			}
			// Use bit is 1: Clear bit and advance hand (second chance)
			f.UseBit = false
			mm.ClockHand = (mm.ClockHand + 1) % mm.NumFrames
		}
	}

	victim := &mm.Frames[victimIdx]
	evictedVPN = victim.VPN

	// If victim page is dirty, it must be written back to swap space
	if victim.IsDirty {
		mm.SwapWrites++
	}

	// Overwrite victim frame with new page
	*victim = FrameInfo{
		PFN:          uint64(victimIdx),
		PID:          pid,
		VPN:          vpn,
		IsOccupied:   true,
		LoadedTick:   mm.CurrentTick,
		LastUsedTick: mm.CurrentTick,
		UseBit:       true,
	}

	return uint64(victimIdx), evictedVPN, true
}

// RenderPhysicalMemory outputs formatted visual table of physical RAM frames
func (mm *MemoryManager) RenderPhysicalMemory() string {
	var sb strings.Builder
	sb.WriteString(visualizer.SubHeader(fmt.Sprintf("Physical RAM Frames (%d Total Frames, Policy: %s)", mm.NumFrames, mm.Policy)))

	tbl := visualizer.NewTable("PFN (Frame)", "PID", "VPN (Page)", "Use Bit (Clock)", "Dirty", "Status")
	tbl.SetAlignment("center", "center", "center", "center", "center", "left")

	for i, f := range mm.Frames {
		if !f.IsOccupied {
			tbl.AddRow(fmt.Sprintf("Frame #%d", i), "-", "-", "-", "-", visualizer.Badge("EMPTY", visualizer.BgBlack, visualizer.FgHiBlack))
			continue
		}

		useStr := "0"
		if f.UseBit {
			useStr = "1"
		}

		dirtyStr := "Clean"
		if f.IsDirty {
			dirtyStr = visualizer.Red("Dirty (Swap Required)")
		}

		pfnLabel := fmt.Sprintf("Frame #%d", i)
		if mm.Policy == PolicyClock && i == mm.ClockHand {
			pfnLabel += visualizer.Cyan(" ◄ [Hand]")
		}

		tbl.AddRow(
			pfnLabel,
			fmt.Sprintf("%d", f.PID),
			fmt.Sprintf("VPN %d", f.VPN),
			useStr,
			dirtyStr,
			visualizer.Badge("OCCUPIED", visualizer.BgGreen, visualizer.FgHiWhite),
		)
	}

	sb.WriteString(tbl.Render())

	stats := []string{
		fmt.Sprintf("Page Faults:       %s%d%s (Pages fetched from disk swap)", visualizer.FgHiYellow, mm.PageFaults, visualizer.Reset),
		fmt.Sprintf("Page Evictions:    %d", mm.PageEvictions),
		fmt.Sprintf("Swap Disk Reads:   %d", mm.SwapReads),
		fmt.Sprintf("Swap Disk Writes:  %d (Dirty page sync)", mm.SwapWrites),
	}
	sb.WriteString("\n" + visualizer.Box("Page Fault & Swap I/O Metrics", stats))

	return sb.String()
}
