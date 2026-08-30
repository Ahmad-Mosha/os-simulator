package kernel

import (
	"fmt"
	"os-simulator/pkg/memory"
	"os-simulator/pkg/visualizer"
	"strings"
)

// IsolationExperimentResult stores output of the memory isolation experiment
type IsolationExperimentResult struct {
	ProcA_PID       int
	ProcB_PID       int
	VirtualAddress  uint64
	ProcA_PFN       uint64
	ProcB_PFN       uint64
	ProcA_Physical  uint64
	ProcB_Physical  uint64
	ProcA_Value     string
	ProcB_Value     string
	CrossAccessTrap bool
	TrapMessage     string
}

// MemoryIsolationLab demonstrates how virtual memory provides hardware-enforced process isolation
type MemoryIsolationLab struct {
	PhysicalRAM map[uint64][]byte // Physical memory indexed by Physical Address
	PageSize    uint64
}

func NewMemoryIsolationLab() *MemoryIsolationLab {
	return &MemoryIsolationLab{
		PhysicalRAM: make(map[uint64][]byte),
		PageSize:    4096,
	}
}

// RunIsolationTest runs a step-by-step experiment proving isolation between two processes
func (lab *MemoryIsolationLab) RunIsolationTest() IsolationExperimentResult {
	vAddr := uint64(0x00401050) // Identical virtual address in both processes
	vpn := vAddr / lab.PageSize
	offset := vAddr % lab.PageSize

	// Process A (PID 100) -> Page Table maps VPN 0x401 to PFN 12
	ptA := memory.NewPageTable(100, 1024, lab.PageSize)
	ptA.MapPage(vpn, 12, true)
	pAddrA := (12 * lab.PageSize) + offset

	// Process B (PID 200) -> Page Table maps SAME VPN 0x401 to PFN 88
	ptB := memory.NewPageTable(200, 1024, lab.PageSize)
	ptB.MapPage(vpn, 88, true)
	pAddrB := (88 * lab.PageSize) + offset

	// 1. Process A writes "Secret_Token_A" into virtual address 0x00401050
	lab.PhysicalRAM[pAddrA] = []byte("Secret_Token_A")

	// 2. Process B writes "Public_Data_B" into the SAME virtual address 0x00401050
	lab.PhysicalRAM[pAddrB] = []byte("Public_Data_B")

	// 3. Process A reads from 0x00401050
	valA := string(lab.PhysicalRAM[pAddrA])

	// 4. Process B reads from 0x00401050
	valB := string(lab.PhysicalRAM[pAddrB])

	// 5. Simulate Process B attempting to access an unmapped / foreign virtual page (e.g. 0x90000000)
	_, _, err := ptB.Translate(0x90000000, false)
	crossAccessTrap := err != nil
	trapMsg := ""
	if err != nil {
		trapMsg = fmt.Sprintf("HARDWARE TRAP (SIGSEGV / #PF): %v", err)
	}

	return IsolationExperimentResult{
		ProcA_PID:       100,
		ProcB_PID:       200,
		VirtualAddress:  vAddr,
		ProcA_PFN:       12,
		ProcB_PFN:       88,
		ProcA_Physical:  pAddrA,
		ProcB_Physical:  pAddrB,
		ProcA_Value:     valA,
		ProcB_Value:     valB,
		CrossAccessTrap: crossAccessTrap,
		TrapMessage:     trapMsg,
	}
}

// RenderIsolationReport visualizes how address translation guarantees process memory isolation
func RenderIsolationReport(res IsolationExperimentResult) string {
	var sb strings.Builder
	sb.WriteString(visualizer.SubHeader("Process Memory Isolation & Virtual Address Space Protection"))

	tbl := visualizer.NewTable("Process", "Virtual Address", "VPN", "PFN (Hardware Frame)", "Physical Address", "Stored Memory Payload")
	tbl.SetAlignment("center", "center", "center", "center", "center", "left")

	tbl.AddRow(
		fmt.Sprintf("Process A (PID %d)", res.ProcA_PID),
		fmt.Sprintf("0x%08X", res.VirtualAddress),
		fmt.Sprintf("0x%04X", res.VirtualAddress/4096),
		fmt.Sprintf("PFN %d", res.ProcA_PFN),
		fmt.Sprintf("0x%08X", res.ProcA_Physical),
		visualizer.Green(fmt.Sprintf("\"%s\"", res.ProcA_Value)),
	)

	tbl.AddRow(
		fmt.Sprintf("Process B (PID %d)", res.ProcB_PID),
		fmt.Sprintf("0x%08X", res.VirtualAddress),
		fmt.Sprintf("0x%04X", res.VirtualAddress/4096),
		fmt.Sprintf("PFN %d", res.ProcB_PFN),
		fmt.Sprintf("0x%08X", res.ProcB_Physical),
		visualizer.Yellow(fmt.Sprintf("\"%s\"", res.ProcB_Value)),
	)

	sb.WriteString(tbl.Render())

	diagram := []string{
		"VIRTUAL ADDRESS ISOLATION PROOF:",
		fmt.Sprintf("• Process A Virtual 0x%08X ──► MMU Translation ──► Physical RAM 0x%08X (\"%s\")", res.VirtualAddress, res.ProcA_Physical, res.ProcA_Value),
		fmt.Sprintf("• Process B Virtual 0x%08X ──► MMU Translation ──► Physical RAM 0x%08X (\"%s\")", res.VirtualAddress, res.ProcB_Physical, res.ProcB_Value),
		"",
		"KEY TAKEAWAYS (OSTEP Chapter 13 & 18):",
		"1. Transparency: Every program believes it has the entire 4GB/64-bit address space to itself.",
		"2. Complete Isolation: Even though both processes use the EXACT same virtual address 0x00401050,",
		"   they are physically separated in hardware RAM. Process B CANNOT see or overwrite Process A's data!",
		"3. Hardware Enforcement: Any attempt to access memory outside a process's page table causes an",
		fmt.Sprintf("   instant Page Fault Exception (#PF / SIGSEGV):\n   %s", visualizer.Red(res.TrapMessage)),
	}

	sb.WriteString("\n" + visualizer.Box("Hardware Isolation Architecture", diagram))
	return sb.String()
}
