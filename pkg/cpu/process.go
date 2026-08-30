package cpu

import (
	"fmt"
)

// ProcessState represents the lifecycle state of a process in the OS
type ProcessState string

const (
	StateNew        ProcessState = "NEW"
	StateReady      ProcessState = "READY"
	StateRunning    ProcessState = "RUNNING"
	StateBlocked    ProcessState = "BLOCKED"
	StateTerminated ProcessState = "TERMINATED"
)

// Registers holds simulated CPU register state saved/restored during context switches
type Registers struct {
	PC    uint64 // Program Counter (Instruction Pointer)
	SP    uint64 // Stack Pointer
	BP    uint64 // Base / Frame Pointer
	AX    uint64 // General Purpose Accumulator
	BX    uint64 // General Purpose Base Register
	CX    uint64 // Counter Register
	DX    uint64 // Data Register
	Flags uint64 // CPU Status Flags (zero, carry, overflow, interrupt enable)
}

// Process (Process Control Block - PCB) represents an OS process
type Process struct {
	PID          int          `json:"pid"`
	Name         string       `json:"name"`
	State        ProcessState `json:"state"`
	Priority     int          `json:"priority"` // Higher value = higher priority (or queue level for MLFQ)
	
	// CPU Burst & Timing metrics (in ticks/ms)
	ArrivalTime   int `json:"arrival_time"`
	BurstTime     int `json:"burst_time"`
	RemainingTime int `json:"remaining_time"`
	StartTime     int `json:"start_time"`    // First time scheduled (-1 if not started)
	FinishTime    int `json:"finish_time"`   // Time when process terminated
	WaitingTime   int `json:"waiting_time"`  // Total time spent in READY state
	Turnaround    int `json:"turnaround"`    // FinishTime - ArrivalTime
	ResponseTime  int `json:"response_time"` // StartTime - ArrivalTime

	// I/O simulation attributes
	IOFrequency int  `json:"io_frequency"` // Performs I/O every N ticks of CPU execution
	IODuration  int  `json:"io_duration"`  // Ticks spent waiting for I/O
	IOCounter   int  `json:"io_counter"`   // Current ticks into I/O burst
	RunTicks    int  `json:"run_ticks"`    // Ticks executed in current CPU burst

	// MLFQ specific tracking (Rule 4: Allotment tracking across interrupts)
	AllotmentUsed int `json:"allotment_used"`
	CurrentQueue  int `json:"current_queue"`

	// Lottery scheduling
	Tickets int `json:"tickets"`

	// Saved Hardware Execution Context (saved to PCB upon descheduling)
	Registers Registers `json:"registers"`

	// Virtual Memory Boundaries (Base and Bounds / Page Table root)
	MemoryBase  uint64 `json:"memory_base"`
	MemoryLimit uint64 `json:"memory_limit"`

	// Associated Threads (TCBs)
	Threads []*Thread `json:"threads,omitempty"`
}

// NewProcess creates a new Process instance in StateNew
func NewProcess(pid int, name string, arrivalTime, burstTime int) *Process {
	return &Process{
		PID:           pid,
		Name:          name,
		State:         StateNew,
		ArrivalTime:   arrivalTime,
		BurstTime:     burstTime,
		RemainingTime: burstTime,
		StartTime:     -1,
		FinishTime:    -1,
		Tickets:       10, // Default 10 lottery tickets
		Registers: Registers{
			PC: 0x00400000, // Typical text segment entry point
			SP: 0x7FFFFFFF, // Top of user stack
		},
	}
}

// Clone creates a fresh deep copy of the process for running in multiple scheduler comparisons
func (p *Process) Clone() *Process {
	clone := *p
	clone.State = StateNew
	clone.RemainingTime = p.BurstTime
	clone.StartTime = -1
	clone.FinishTime = -1
	clone.WaitingTime = 0
	clone.Turnaround = 0
	clone.ResponseTime = 0
	clone.AllotmentUsed = 0
	clone.CurrentQueue = 0
	clone.RunTicks = 0
	clone.IOCounter = 0
	return &clone
}

// String returns a clean summary string of the process
func (p *Process) String() string {
	return fmt.Sprintf("PID:%d [%s] State:%s Arr:%d Burst:%d Rem:%d Wait:%d Turnaround:%d",
		p.PID, p.Name, p.State, p.ArrivalTime, p.BurstTime, p.RemainingTime, p.WaitingTime, p.Turnaround)
}
