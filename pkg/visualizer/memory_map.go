package visualizer

import (
	"fmt"
	"strings"
)

// RenderAddressSpaceLayout creates an ASCII diagram of a process virtual address space
func RenderAddressSpaceLayout(maxAddr uint64, stackTop, stackBottom, heapBottom, heapTop, dataEnd, textEnd uint64) string {
	var sb strings.Builder

	sb.WriteString(SubHeader("Process Virtual Address Space (OSTEP Chapter 13)"))
	sb.WriteString(fmt.Sprintf("  %s0x%08X%s ┌───────────────────────────────────────┐  %sHigh Memory%s\n", FgHiBlack, maxAddr, Reset, FgHiBlack, Reset))
	sb.WriteString(fmt.Sprintf("             │  %sKernel Space (Protected)%s            │\n", FgHiBlack, Reset))
	sb.WriteString(fmt.Sprintf("  %s0x%08X%s ├───────────────────────────────────────┤\n", FgCyan, stackTop, Reset))
	sb.WriteString(fmt.Sprintf("             │  %sStack (grows DOWN ↓)%s                │  Local variables, return addrs\n", Bold+FgHiGreen, Reset))
	sb.WriteString(fmt.Sprintf("  %s0x%08X%s │  %s↑ Stack Pointer (SP)%s               │\n", FgGreen, stackBottom, Reset, FgGreen, Reset))
	sb.WriteString("             │                  ↓                    │\n")
	sb.WriteString("             │             (Unallocated)             │\n")
	sb.WriteString("             │                  ↑                    │\n")
	sb.WriteString(fmt.Sprintf("  %s0x%08X%s │  %s↑ Program Break (brk)%s               │\n", FgYellow, heapTop, Reset, FgYellow, Reset))
	sb.WriteString(fmt.Sprintf("             │  %sHeap (grows UP ↑)%s                  │  Dynamic malloc / make / new\n", Bold+FgHiYellow, Reset))
	sb.WriteString(fmt.Sprintf("  %s0x%08X%s ├───────────────────────────────────────┤\n", FgCyan, heapBottom, Reset))
	sb.WriteString(fmt.Sprintf("             │  %sData & BSS Segment%s                 │  Globals & static variables\n", Bold+FgHiMagenta, Reset))
	sb.WriteString(fmt.Sprintf("  %s0x%08X%s ├───────────────────────────────────────┤\n", FgCyan, dataEnd, Reset))
	sb.WriteString(fmt.Sprintf("             │  %sCode / Text Segment (Read-Only)%s    │  Executable machine instructions\n", Bold+FgHiCyan, Reset))
	sb.WriteString(fmt.Sprintf("  %s0x00000000%s └───────────────────────────────────────┘  %sLow Memory (0x0)%s\n", FgCyan, Reset, FgHiBlack, Reset))

	return sb.String()
}

// RenderTranslationDiagram displays the bitwise decomposition of virtual address to physical address
func RenderTranslationDiagram(vAddr uint64, vpn, offset, pfn, pAddr uint64, tlbHit bool) string {
	var sb strings.Builder

	status := Badge("TLB HIT", BgGreen, FgHiWhite)
	if !tlbHit {
		status = Badge("TLB MISS (Page Table Walk)", BgRed, FgHiWhite)
	}

	sb.WriteString(fmt.Sprintf("\n%s\n", status))
	sb.WriteString(fmt.Sprintf("  Virtual Address:   %s0x%08X%s (Binary: %s)\n", FgHiCyan, vAddr, Reset, fmt.Sprintf("%032b", vAddr)))
	sb.WriteString(fmt.Sprintf("  ├── VPN (Page Num): %s0x%04X%s (%d)\n", FgHiYellow, vpn, Reset, vpn))
	sb.WriteString(fmt.Sprintf("  └── Offset:         %s0x%04X%s (%d bytes into page)\n", FgHiMagenta, offset, Reset, offset))
	sb.WriteString("          │\n")
	sb.WriteString(fmt.Sprintf("          ▼ Translation (Page Table / TLB): VPN %d ──► PFN %d\n", vpn, pfn))
	sb.WriteString("          │\n")
	sb.WriteString(fmt.Sprintf("  Physical Address:  %s0x%08X%s [PFN: 0x%04X | Offset: 0x%04X]\n", FgHiGreen, pAddr, Reset, pfn, offset))

	return sb.String()
}
