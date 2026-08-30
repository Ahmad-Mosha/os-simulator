package cpu

import (
	"fmt"
	"os-simulator/pkg/visualizer"
	"strings"
)

// ContextSwitchType defines whether the switch is between threads in same process or different processes
type ContextSwitchType string

const (
	SwitchProcess ContextSwitchType = "PROCESS_SWITCH" // Expensive: full register save/restore + address space switch + TLB flush
	SwitchThread  ContextSwitchType = "THREAD_SWITCH"  // Cheaper: register save/restore + stack pointer switch, same page table
)

// SwitchRecord logs a single context switch event for educational inspection
type SwitchRecord struct {
	Timestamp      int               `json:"timestamp"`
	Type           ContextSwitchType `json:"type"`
	FromEntity     string            `json:"from"`
	ToEntity       string            `json:"to"`
	CyclesCost     int               `json:"cycles_cost"`
	TLBFlushed     bool              `json:"tlb_flushed"`
	RegistersSaved Registers         `json:"saved_registers"`
}

// ContextSwitcher manages hardware-level context switching simulations
type ContextSwitcher struct {
	TotalSwitches   int
	TotalCyclesCost int
	History         []SwitchRecord
	
	// Cost constants (in simulated CPU cycles)
	ThreadSwitchCycles  int // Typically ~1,000 cycles (~0.5-1µs)
	ProcessSwitchCycles int // Typically ~5,000-10,000 cycles (~2-5µs due to TLB invalidation & cache misses)
}

// NewContextSwitcher initializes a switcher with realistic cycle overheads
func NewContextSwitcher() *ContextSwitcher {
	return &ContextSwitcher{
		ThreadSwitchCycles:  1000,
		ProcessSwitchCycles: 6000,
		History:             make([]SwitchRecord, 0),
	}
}

// SwitchProcessContext performs a simulated context switch between two processes
func (cs *ContextSwitcher) SwitchProcessContext(time int, from, to *Process) SwitchRecord {
	cs.TotalSwitches++
	cs.TotalCyclesCost += cs.ProcessSwitchCycles

	var savedRegs Registers
	fromName := "IDLE"
	if from != nil {
		fromName = fmt.Sprintf("P%d(%s)", from.PID, from.Name)
		from.State = StateReady
		// Simulate saving CPU registers into outgoing PCB
		from.Registers.PC += 4 // Advanced instruction pointer
		savedRegs = from.Registers
	}

	toName := "IDLE"
	if to != nil {
		toName = fmt.Sprintf("P%d(%s)", to.PID, to.Name)
		to.State = StateRunning
		if to.StartTime == -1 {
			to.StartTime = time
			to.ResponseTime = time - to.ArrivalTime
		}
	}

	rec := SwitchRecord{
		Timestamp:      time,
		Type:           SwitchProcess,
		FromEntity:     fromName,
		ToEntity:       toName,
		CyclesCost:     cs.ProcessSwitchCycles,
		TLBFlushed:     true, // Changing address spaces invalidates virtual-to-physical translations
		RegistersSaved: savedRegs,
	}

	cs.History = append(cs.History, rec)
	return rec
}

// ExplainContextSwitch returns a step-by-step educational breakdown of what occurs during a context switch
func (cs *ContextSwitcher) ExplainContextSwitch(rec SwitchRecord) string {
	var sb strings.Builder

	sb.WriteString(visualizer.SubHeader(fmt.Sprintf("Context Switch Event @ Tick %d [%s -> %s]", rec.Timestamp, rec.FromEntity, rec.ToEntity)))
	
	lines := []string{
		fmt.Sprintf("1. Hardware Interrupt (Timer / Yield) fires: CPU switches to Kernel Mode (Ring 0)"),
		fmt.Sprintf("2. Save outgoing context: Registers (PC=0x%X, SP=0x%X) pushed to Kernel Stack & PCB", rec.RegistersSaved.PC, rec.RegistersSaved.SP),
		fmt.Sprintf("3. OS Scheduler chooses next entity: %s", rec.ToEntity),
	}

	if rec.Type == SwitchProcess {
		lines = append(lines,
			"4. Virtual Memory Switch: Update CR3 / TTBR0 register to point to incoming process Page Table",
			"5. Hardware Cache Impact: TLB (Translation Lookaside Buffer) flushed / ASID updated",
			fmt.Sprintf("6. Cost: ~%d CPU cycles (Heavy due to memory cache/TLB misses)", rec.CyclesCost),
		)
	} else {
		lines = append(lines,
			"4. Address Space Remains UNCHANGED (Threads share Page Directory / Heap)",
			"5. Switch Stack Pointer (SP) and Thread Control Block (TCB)",
			fmt.Sprintf("6. Cost: ~%d CPU cycles (Lightweight)", rec.CyclesCost),
		)
	}

	lines = append(lines, "7. Restore incoming registers: Execute 'iret' / return from trap to User Mode (Ring 3)")

	sb.WriteString(visualizer.Box("OS Mechanism Under The Hood", lines))
	return sb.String()
}
