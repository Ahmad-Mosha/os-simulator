package kernel

import (
	"fmt"
	"os-simulator/pkg/visualizer"
	"strings"
)

// TrapType distinguishes software traps, CPU exceptions, and hardware interrupts
type TrapType string

const (
	TrapSyscall   TrapType = "SOFTWARE_TRAP"     // Initiated intentionally by user code (e.g. INT 0x80, SYSCALL)
	TrapException TrapType = "CPU_EXCEPTION"     // Synchronous error caused by current instruction (e.g. Divide by Zero, Page Fault)
	TrapInterrupt TrapType = "HARDWARE_INTERRUPT"// Asynchronous hardware event (e.g. Timer tick, Keyboard keypress, Disk I/O ready)
)

// TrapHandlerFunc defines signature for an Interrupt Service Routine (ISR)
type TrapHandlerFunc func(cpu *HardwareCPU, ctx *TrapContext) (status int, err error)

// TrapContext holds state saved on kernel stack upon trap entry
type TrapContext struct {
	Vector       int
	Type         TrapType
	Name         string
	SavedPC      uint64
	SavedRSP     uint64
	SavedFlags   uint64
	ErrorCode    uint64
	SyscallNum   int
	Args         [6]uint64
	ReturnValue  uint64
}

// TrapDescriptor represents a single gate in the Interrupt Descriptor Table (IDT)
type TrapDescriptor struct {
	Vector      int
	Name        string
	Type        TrapType
	Description string
	Handler     TrapHandlerFunc
	Count       int
}

// TrapTable simulates the CPU Interrupt Descriptor Table (IDT)
type TrapTable struct {
	Descriptors map[int]*TrapDescriptor
	History     []TrapContext
}

func NewTrapTable() *TrapTable {
	tt := &TrapTable{
		Descriptors: make(map[int]*TrapDescriptor),
		History:     make([]TrapContext, 0),
	}

	// 1. Register Standard CPU Exceptions (Vectors 0-31)
	tt.RegisterHandler(0, "Divide-by-Zero (#DE)", TrapException, "Triggered by integer division by zero", defaultExceptionHandler)
	tt.RegisterHandler(6, "Invalid Opcode (#UD)", TrapException, "CPU encountered unmapped instruction opcode", defaultExceptionHandler)
	tt.RegisterHandler(13, "General Protection Fault (#GP)", TrapException, "Privilege violation or memory segment fault", defaultExceptionHandler)
	tt.RegisterHandler(14, "Page Fault (#PF)", TrapException, "Page not present in RAM or protection violation (CR2 holds faulting address)", defaultExceptionHandler)

	// 2. Register Hardware Interrupts (Vectors 32-255)
	tt.RegisterHandler(32, "PIT Timer Interrupt (IRQ 0)", TrapInterrupt, "Periodic hardware timer for preemptive CPU scheduling", defaultInterruptHandler)
	tt.RegisterHandler(33, "Keyboard Controller (IRQ 1)", TrapInterrupt, "Key press / release event", defaultInterruptHandler)
	tt.RegisterHandler(46, "Disk Controller (IRQ 14)", TrapInterrupt, "Disk read/write DMA transfer completed", defaultInterruptHandler)

	// 3. Register Software Traps / Syscall Gate (Vector 0x80 / 128)
	tt.RegisterHandler(0x80, "System Call Gate (INT 0x80)", TrapSyscall, "User space requesting kernel service", defaultSyscallGateHandler)

	return tt
}

// RegisterHandler configures an entry in the IDT
func (tt *TrapTable) RegisterHandler(vector int, name string, tType TrapType, desc string, handler TrapHandlerFunc) {
	tt.Descriptors[vector] = &TrapDescriptor{
		Vector:      vector,
		Name:        name,
		Type:        tType,
		Description: desc,
		Handler:     handler,
	}
}

// DispatchTrap processes a trap/interrupt: switches CPU mode, saves context, executes ISR, returns
func (tt *TrapTable) DispatchTrap(cpu *HardwareCPU, vector int, syscallNum int, args [6]uint64) (*TrapContext, error) {
	desc, exists := tt.Descriptors[vector]
	if !exists {
		return nil, fmt.Errorf("unhandled trap vector 0x%02X: no ISR registered in IDT", vector)
	}

	desc.Count++

	// 1. Hardware auto-saves user state & switches to Kernel Mode
	ctx := TrapContext{
		Vector:     vector,
		Type:       desc.Type,
		Name:       desc.Name,
		SavedPC:    cpu.ProgramCounter,
		SavedRSP:   cpu.UserRSP,
		SyscallNum: syscallNum,
		Args:       args,
	}

	cpu.SwitchToKernelMode(vector)

	// 2. Execute Interrupt Service Routine (ISR)
	if desc.Handler != nil {
		_, err := desc.Handler(cpu, &ctx)
		if err != nil {
			// Trap handled with error
		}
	}

	// 3. Return from trap (IRET / SYSRET)
	cpu.ReturnToUserMode()

	tt.History = append(tt.History, ctx)
	return &ctx, nil
}

func defaultExceptionHandler(cpu *HardwareCPU, ctx *TrapContext) (int, error) {
	return -1, fmt.Errorf("CPU Exception %s triggered at PC:0x%X", ctx.Name, ctx.SavedPC)
}

func defaultInterruptHandler(cpu *HardwareCPU, ctx *TrapContext) (int, error) {
	return 0, nil
}

func defaultSyscallGateHandler(cpu *HardwareCPU, ctx *TrapContext) (int, error) {
	ctx.ReturnValue = 0 // Success
	return 0, nil
}

// RenderIDT visualizes the Interrupt Descriptor Table
func (tt *TrapTable) RenderIDT() string {
	var sb strings.Builder
	sb.WriteString(visualizer.SubHeader("Interrupt Descriptor Table (IDT / Trap Vector Table)"))

	tbl := visualizer.NewTable("Vector", "Name", "Classification", "Description", "Fired Count")
	tbl.SetAlignment("center", "left", "center", "left", "right")

	vectors := []int{0, 6, 13, 14, 32, 33, 46, 0x80}
	for _, v := range vectors {
		if desc, exists := tt.Descriptors[v]; exists {
			typeBadge := visualizer.Badge("EXCEPTION", visualizer.BgRed, visualizer.FgHiWhite)
			if desc.Type == TrapInterrupt {
				typeBadge = visualizer.Badge("HARDWARE IRQ", visualizer.BgYellow, visualizer.FgHiWhite)
			} else if desc.Type == TrapSyscall {
				typeBadge = visualizer.Badge("SYSCALL GATE", visualizer.BgGreen, visualizer.FgHiWhite)
			}

			tbl.AddRow(
				fmt.Sprintf("0x%02X (%d)", v, v),
				desc.Name,
				typeBadge,
				desc.Description,
				fmt.Sprintf("%d", desc.Count),
			)
		}
	}

	sb.WriteString(tbl.Render())

	deepDive := []string{
		"TRAP / INTERRUPT DISPATCH MECHANISM (Hardware Step-by-Step):",
		"1. Trigger: Software (INT 0x80/SYSCALL), Hardware Device (Timer/NIC), or CPU Error (Div0/PageFault).",
		"2. Hardware Context Save: CPU hardware pushes SS, RSP, RFLAGS, CS, RIP onto the private KERNEL STACK.",
		"3. Privilege Transition: CPU clears CPL (Current Privilege Level) to Ring 0 (Kernel Mode).",
		"4. Vector Lookup: CPU indexes the IDT at 'IDTR_Base + (Vector * 16)' to find the ISR function address.",
		"5. ISR Execution: Kernel executes C/Go handler code.",
		"6. IRET / SYSRET: Hardware pops saved registers from kernel stack and restores User Mode (Ring 3)!",
	}
	sb.WriteString("\n" + visualizer.Box("Hardware Trap & Interrupt Sequence", deepDive))

	return sb.String()
}
