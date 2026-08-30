package cpu

import "fmt"

// ThreadState represents lifecycle states of an OS thread
type ThreadState string

const (
	ThreadStateReady      ThreadState = "READY"
	ThreadStateRunning    ThreadState = "RUNNING"
	ThreadStateBlocked    ThreadState = "BLOCKED"
	ThreadStateTerminated ThreadState = "TERMINATED"
)

// Thread (Thread Control Block - TCB) represents an execution context within a Process
// KEY OS CONCEPT:
// A Process is an address space + resource container (Code, Data, Heap, File Descriptors).
// A Thread is an independent stream of instruction execution sharing the Process's address space,
// but having its OWN:
//  1. Thread ID (TID)
//  2. Register set (PC, SP, general purpose registers)
//  3. Private Stack (for local variables, function calls, return addresses)
type Thread struct {
	TID       int         `json:"tid"`
	PID       int         `json:"pid"`
	Name      string      `json:"name"`
	State     ThreadState `json:"state"`
	Registers Registers   `json:"registers"`

	// Thread-Private Stack boundaries within the process address space
	StackBase  uint64 `json:"stack_base"`
	StackLimit uint64 `json:"stack_limit"`
	StackSize  uint64 `json:"stack_size"` // e.g. 8MB on Linux / 2MB default
}

// NewThread initializes a new thread inside a parent process
func NewThread(tid, pid int, name string, stackBase, stackSize uint64) *Thread {
	return &Thread{
		TID:        tid,
		PID:        pid,
		Name:       name,
		State:      ThreadStateReady,
		StackBase:  stackBase,
		StackSize:  stackSize,
		StackLimit: stackBase - stackSize, // Stacks grow downwards
		Registers: Registers{
			PC: 0x00401000,
			SP: stackBase,
			BP: stackBase,
		},
	}
}

func (t *Thread) String() string {
	return fmt.Sprintf("TID:%d (PID:%d) [%s] State:%s SP:0x%08X Stack:[0x%08X-0x%08X]",
		t.TID, t.PID, t.Name, t.State, t.Registers.SP, t.StackLimit, t.StackBase)
}
