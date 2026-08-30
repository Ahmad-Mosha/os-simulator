package kernel

import (
	"errors"
	"fmt"
	"os-simulator/pkg/visualizer"
	"strings"
)

var (
	ErrPrivilegeViolation = errors.New("privilege violation: user mode cannot execute privileged CPU instructions")
	ErrKernelMemoryAccess = errors.New("protection violation: user mode cannot access kernel memory space (Ring 0)")
)

// CPUMode represents hardware CPU execution privilege level (x86 Rings)
type CPUMode int

const (
	ModeKernel CPUMode = 0 // Ring 0: Full hardware access, direct I/O, page table control
	ModeUser   CPUMode = 3 // Ring 3: Restricted execution, virtual memory bounds, no direct hardware I/O
)

func (m CPUMode) String() string {
	if m == ModeKernel {
		return "KERNEL_MODE (Ring 0)"
	}
	return "USER_MODE (Ring 3)"
}

// InstructionType categorizes CPU instructions by required privilege level
type InstructionType string

const (
	InstUserArithmetic   InstructionType = "USER_ARITHMETIC"    // ADD, SUB, MUL, XOR, MOV (User RAM)
	InstUserBranch       InstructionType = "USER_BRANCH"        // JMP, JZ, CALL, RET
	InstSyscall          InstructionType = "SYSCALL_TRAP"       // INT 0x80, SYSCALL (Transitions to Ring 0)
	InstPrivilegedHalt   InstructionType = "PRIV_HLT"           // HLT (Stops CPU until interrupt)
	InstPrivilegedCLI    InstructionType = "PRIV_CLI"           // CLI (Disable hardware interrupts)
	InstPrivilegedSTI    InstructionType = "PRIV_STI"           // STI (Enable hardware interrupts)
	InstPrivilegedSetCR3 InstructionType = "PRIV_MOV_CR3"       // Set Page Directory Base Register
	InstPrivilegedPortIO InstructionType = "PRIV_PORT_IO"       // IN / OUT direct device bus I/O
)

// CPUInstruction represents a simulated machine instruction
type CPUInstruction struct {
	Opcode       InstructionType
	Description  string
	IsPrivileged bool
}

// HardwareCPU simulates dual-mode CPU execution and Ring 3 -> Ring 0 transitions
type HardwareCPU struct {
	CurrentMode    CPUMode
	UserRSP        uint64 // User Stack Pointer
	KernelRSP      uint64 // Kernel Stack Pointer (swapped during Ring transition)
	ProgramCounter uint64 // RIP / PC
	CR3            uint64 // Page Table Base Register
	InterruptsOn   bool
	ExecutionLog   []string
}

func NewHardwareCPU() *HardwareCPU {
	return &HardwareCPU{
		CurrentMode:    ModeUser,
		UserRSP:        0x7FFFFFFF,
		KernelRSP:      0xFFFFFFFF,
		ProgramCounter: 0x00400000,
		CR3:            0x00100000,
		InterruptsOn:   true,
		ExecutionLog:   make([]string, 0),
	}
}

// ExecuteInstruction attempts to run an instruction under the current CPU mode
func (cpu *HardwareCPU) ExecuteInstruction(inst CPUInstruction) error {
	logMsg := fmt.Sprintf("[%s] PC:0x%08X -> Executing: %s (%s)",
		cpu.CurrentMode, cpu.ProgramCounter, inst.Opcode, inst.Description)

	if inst.IsPrivileged && cpu.CurrentMode == ModeUser {
		// Hardware General Protection Fault (#GP, Trap Vector 13)
		cpu.ExecutionLog = append(cpu.ExecutionLog, logMsg+" ──► "+visualizer.Red("HARDWARE TRAP: PRIVILEGE VIOLATION (#GP)!"))
		return fmt.Errorf("%w: attempt to execute '%s' in User Mode (Ring 3)", ErrPrivilegeViolation, inst.Opcode)
	}

	// Instruction succeeds
	switch inst.Opcode {
	case InstPrivilegedCLI:
		cpu.InterruptsOn = false
	case InstPrivilegedSTI:
		cpu.InterruptsOn = true
	case InstPrivilegedSetCR3:
		cpu.CR3 = 0x00200000 // Update page table base
	case InstSyscall:
		// Transition to Kernel Mode
		cpu.SwitchToKernelMode(0x80)
	}

	cpu.ProgramCounter += 4
	cpu.ExecutionLog = append(cpu.ExecutionLog, logMsg+" ──► "+visualizer.Green("SUCCESS"))
	return nil
}

// SwitchToKernelMode simulates hardware trap transition from User (Ring 3) to Kernel (Ring 0)
func (cpu *HardwareCPU) SwitchToKernelMode(vector int) {
	if cpu.CurrentMode == ModeKernel {
		return
	}
	cpu.CurrentMode = ModeKernel
	// Hardware automatically switches RSP to Kernel Stack
	cpu.ExecutionLog = append(cpu.ExecutionLog, fmt.Sprintf(
		"[HARDWARE TRAP 0x%X] Privilege Switch: Ring 3 ──► Ring 0 | Swapped RSP: User(0x%X) ──► Kernel(0x%X)",
		vector, cpu.UserRSP, cpu.KernelRSP))
}

// ReturnToUserMode simulates IRET / SYSRET instruction returning to User Mode (Ring 3)
func (cpu *HardwareCPU) ReturnToUserMode() {
	if cpu.CurrentMode == ModeUser {
		return
	}
	cpu.CurrentMode = ModeUser
	cpu.ExecutionLog = append(cpu.ExecutionLog, fmt.Sprintf(
		"[SYSRET/IRET] Privilege Switch: Ring 0 ──► Ring 3 | Restored RSP: 0x%X", cpu.UserRSP))
}

// RenderModeComparison visualizes differences between User Mode and Kernel Mode
func RenderModeComparison() string {
	var sb strings.Builder
	sb.WriteString(visualizer.SubHeader("CPU Privilege Rings: User Mode (Ring 3) vs Kernel Mode (Ring 0)"))

	tbl := visualizer.NewTable("Dimension", "User Mode (Ring 3)", "Kernel Mode (Ring 0)")
	tbl.SetAlignment("left", "left", "left")

	tbl.AddRow("Privilege Level", "Lowest (Restricted)", "Highest (Full Supervisor)")
	tbl.AddRow("Memory Access", "Virtual address space (User segments only)", "All Physical & Virtual memory (Direct)")
	tbl.AddRow("Privileged Instructions", visualizer.Badge("BLOCKED (Traps #GP)", visualizer.BgRed, visualizer.FgHiWhite), visualizer.Badge("ALLOWED (Full)", visualizer.BgGreen, visualizer.FgHiWhite))
	tbl.AddRow("I/O Device Access", "Restricted (Must request via Syscall)", "Direct Hardware Bus (IN / OUT, MMIO)")
	tbl.AddRow("Interrupt Control", "Cannot disable interrupts", "Can disable/enable interrupts (CLI / STI)")
	tbl.AddRow("Stack Used", "User Space Stack", "Private Kernel Stack (Safe from user corruption)")

	sb.WriteString(tbl.Render())

	deepDive := []string{
		"WHY DUAL-MODE CPU OPERATION IS CRITICAL:",
		"1. Crash Containment & Protection:",
		"   - If a user application has a bug (e.g. null pointer dereference), only that process dies.",
		"   - User code CANNOT corrupt other applications or the operating system kernel.",
		"2. Security Boundary:",
		"   - User code cannot read passwords from raw physical RAM or eavesdrop on network packets without kernel permission.",
		"3. Transition Mechanism:",
		"   - User applications enter Kernel Mode ONLY through controlled gates: System Calls (INT 0x80 / SYSCALL) and Traps.",
	}
	sb.WriteString("\n" + visualizer.Box("Dual-Mode OS Security & Stability", deepDive))

	return sb.String()
}
