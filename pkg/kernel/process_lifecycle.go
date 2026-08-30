package kernel

import (
	"errors"
	"fmt"
	"os-simulator/pkg/memory"
	"os-simulator/pkg/visualizer"
	"strings"
	"sync"
)

var (
	ErrNoSuchProcess = errors.New("no such process (ESRCH)")
	ErrNoChild       = errors.New("no child processes (ECHILD)")
)

// UnixProcessState defines the Unix process state machine
type UnixProcessState string

const (
	ProcStateReady   UnixProcessState = "READY"
	ProcStateRunning UnixProcessState = "RUNNING"
	ProcStateBlocked UnixProcessState = "BLOCKED"
	ProcStateZombie  UnixProcessState = "ZOMBIE" // Exited, holding exit status for parent wait()
	ProcStateReaped  UnixProcessState = "REAPED" // Reaped by parent, PCB destroyed
)

// UnixProcess represents a full Unix process with parent/child relationships and address space
type UnixProcess struct {
	PID          int
	PPID         int // Parent PID
	Name         string
	State        UnixProcessState
	ExitCode     int
	AddressSpace *memory.AddressSpace
	PageTable    *memory.PageTable
	Registers    SyscallRegisters
	OpenFDs      map[int]string // Simulated File Descriptor table
	Children     []int          // List of Child PIDs
}

// ProcessManager coordinates Unix process creation, execution, and termination
type ProcessManager struct {
	Processes   map[int]*UnixProcess
	NextPID     int
	InitPID     int
	LifecycleLog []string
	mu          sync.Mutex
}

func NewProcessManager() *ProcessManager {
	pm := &ProcessManager{
		Processes:    make(map[int]*UnixProcess),
		NextPID:      1,
		InitPID:      1,
		LifecycleLog: make([]string, 0),
	}

	// Create Init Process (PID 1) - Root ancestor
	initProc := &UnixProcess{
		PID:          1,
		PPID:         0,
		Name:         "init",
		State:        ProcStateReady,
		AddressSpace: memory.NewAddressSpace(1, 64*1024, 0x00100000),
		PageTable:    memory.NewPageTable(1, 16, 4096),
		OpenFDs:      map[int]string{0: "stdin", 1: "stdout", 2: "stderr"},
		Children:     make([]int, 0),
	}
	pm.Processes[1] = initProc
	pm.NextPID = 2

	return pm
}

// Fork duplicates the calling process (OSTEP Chapter 5: Process API)
// Returns:
// In Parent: Child PID
// In Child: 0
func (pm *ProcessManager) Fork(parentPID int) (*UnixProcess, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	parent, exists := pm.Processes[parentPID]
	if !exists {
		return nil, fmt.Errorf("%w: PID %d", ErrNoSuchProcess, parentPID)
	}

	childPID := pm.NextPID
	pm.NextPID++

	// Duplicate open file descriptors
	childFDs := make(map[int]string)
	for fd, target := range parent.OpenFDs {
		childFDs[fd] = target
	}

	// Clone virtual address space
	childAS := memory.NewAddressSpace(childPID, parent.AddressSpace.TotalSize, parent.AddressSpace.PhysicalBase+uint64(childPID*0x10000))
	childPT := memory.NewPageTable(childPID, 16, 4096)
	for i, entry := range parent.PageTable.Entries {
		if entry.Valid {
			childPT.MapPage(uint64(i), entry.PFN+uint64(childPID*10), entry.ReadWrite)
		}
	}

	child := &UnixProcess{
		PID:          childPID,
		PPID:         parentPID,
		Name:         fmt.Sprintf("%s_child", parent.Name),
		State:        ProcStateReady,
		AddressSpace: childAS,
		PageTable:    childPT,
		Registers:    parent.Registers,
		OpenFDs:      childFDs,
		Children:     make([]int, 0),
	}

	// Unix Fork Semantics: Child receives return value 0 in RAX
	child.Registers.RAX = 0

	// Parent receives child PID in RAX
	parent.Registers.RAX = uint64(childPID)
	parent.Children = append(parent.Children, childPID)

	pm.Processes[childPID] = child

	logMsg := fmt.Sprintf("[FORK] Parent P%d (%s) cloned child P%d -> Parent RAX=%d, Child RAX=0",
		parentPID, parent.Name, childPID, childPID)
	pm.LifecycleLog = append(pm.LifecycleLog, logMsg)

	return child, nil
}

// Exec replaces the calling process's memory space with a new executable program
func (pm *ProcessManager) Exec(pid int, binaryName string, args []string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	proc, exists := pm.Processes[pid]
	if !exists {
		return fmt.Errorf("%w: PID %d", ErrNoSuchProcess, pid)
	}

	oldName := proc.Name
	proc.Name = binaryName

	// Replace Address Space with fresh memory layout
	proc.AddressSpace = memory.NewAddressSpace(pid, 64*1024, proc.AddressSpace.PhysicalBase)
	proc.PageTable = memory.NewPageTable(pid, 16, 4096)
	proc.PageTable.MapPage(0, uint64(pid*20), true)

	// Reset instruction pointer to new program entry point
	proc.Registers.RAX = 0
	proc.Registers.RDI = uint64(len(args)) // argc
	proc.Registers.RSI = 0x00408000        // argv pointer

	logMsg := fmt.Sprintf("[EXEC] P%d replaced program image '%s' ──► '%s' (Preserved PID %d, Reset Address Space)",
		pid, oldName, binaryName, pid)
	pm.LifecycleLog = append(pm.LifecycleLog, logMsg)

	return nil
}

// Exit terminates the process, turning it into a ZOMBIE until parent calls wait()
func (pm *ProcessManager) Exit(pid int, exitCode int) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	proc, exists := pm.Processes[pid]
	if !exists {
		return fmt.Errorf("%w: PID %d", ErrNoSuchProcess, pid)
	}

	if pid == pm.InitPID {
		return fmt.Errorf("kernel panic: attempt to terminate init process (PID 1)")
	}

	proc.State = ProcStateZombie
	proc.ExitCode = exitCode

	// Free virtual address space memory (only PCB and exit code remain)
	proc.AddressSpace = nil

	// ORPHAN REPARENTING: If this process has children, reparent them to init (PID 1)
	if len(proc.Children) > 0 {
		initProc := pm.Processes[pm.InitPID]
		for _, childPID := range proc.Children {
			if child, ok := pm.Processes[childPID]; ok {
				child.PPID = pm.InitPID
				initProc.Children = append(initProc.Children, childPID)
				pm.LifecycleLog = append(pm.LifecycleLog, fmt.Sprintf(
					"[ORPHAN REPARENTED] Child P%d adopted by init (PID 1) after parent P%d exited", childPID, pid))
			}
		}
		proc.Children = make([]int, 0)
	}

	logMsg := fmt.Sprintf("[EXIT] Process P%d (%s) exited with code %d ──► Entered %s state",
		pid, proc.Name, exitCode, visualizer.Badge("ZOMBIE", visualizer.BgYellow, visualizer.FgHiWhite))
	pm.LifecycleLog = append(pm.LifecycleLog, logMsg)

	return nil
}

// Wait reaps a zombie child process and retrieves its exit code
func (pm *ProcessManager) Wait(parentPID int) (reapedPID int, exitCode int, err error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	parent, exists := pm.Processes[parentPID]
	if !exists {
		return 0, 0, fmt.Errorf("%w: PID %d", ErrNoSuchProcess, parentPID)
	}

	if len(parent.Children) == 0 {
		return 0, 0, fmt.Errorf("%w: parent PID %d has no child processes", ErrNoChild, parentPID)
	}

	// Search for a child in ZOMBIE state
	for i, childPID := range parent.Children {
		child, ok := pm.Processes[childPID]
		if ok && child.State == ProcStateZombie {
			// REAP ZOMBIE PROCESS: Read exit code, delete PCB
			exitCode = child.ExitCode
			reapedPID = childPID
			child.State = ProcStateReaped
			delete(pm.Processes, childPID)

			// Remove from parent's children list
			parent.Children = append(parent.Children[:i], parent.Children[i+1:]...)

			logMsg := fmt.Sprintf("[WAIT/REAP] Parent P%d reaped Zombie child P%d (Exit Code: %d) ──► PCB destroyed",
				parentPID, childPID, exitCode)
			pm.LifecycleLog = append(pm.LifecycleLog, logMsg)

			return reapedPID, exitCode, nil
		}
	}

	return 0, 0, fmt.Errorf("EWOULDBLOCK: child processes are still running (parent must block)")
}

// RenderProcessTree outputs visual table of active processes and zombies
func (pm *ProcessManager) RenderProcessTree() string {
	var sb strings.Builder
	sb.WriteString(visualizer.SubHeader("Unix Process Table & Lifecycle State"))

	tbl := visualizer.NewTable("PID", "PPID", "Process Name", "State", "Exit Code", "Open Descriptors", "Children")
	tbl.SetAlignment("center", "center", "left", "center", "center", "left", "left")

	for _, p := range pm.Processes {
		badge := visualizer.Badge(string(p.State), visualizer.BgGreen, visualizer.FgHiWhite)
		if p.State == ProcStateZombie {
			badge = visualizer.Badge("ZOMBIE", visualizer.BgYellow, visualizer.FgHiWhite)
		}

		exitStr := "-"
		if p.State == ProcStateZombie {
			exitStr = fmt.Sprintf("%d", p.ExitCode)
		}

		fds := make([]string, 0)
		for fd, name := range p.OpenFDs {
			fds = append(fds, fmt.Sprintf("%d:%s", fd, name))
		}

		children := fmt.Sprintf("%v", p.Children)
		if len(p.Children) == 0 {
			children = "none"
		}

		tbl.AddRow(
			fmt.Sprintf("%d", p.PID),
			fmt.Sprintf("%d", p.PPID),
			p.Name,
			badge,
			exitStr,
			strings.Join(fds, ", "),
			children,
		)
	}

	sb.WriteString(tbl.Render())

	theory := []string{
		"UNIX PROCESS LIFECYCLE (OSTEP Chapter 5):",
		"1. fork(): Duplicates calling process. Returns 0 to child, child PID to parent.",
		"2. execve(): Replaces process code/data/stack with a new executable (retains PID & FDs).",
		"3. exit(): Releases memory, sets state to ZOMBIE. Zombie PCB remains to store exit code.",
		"4. wait() / waitpid(): Parent reads child exit code and REAPS zombie PCB.",
		"5. Zombie Process: Terminated process whose parent has not yet called wait().",
		"6. Orphan Process: Process whose parent died; automatically adopted and reaped by init (PID 1).",
	}
	sb.WriteString("\n" + visualizer.Box("Process Lifecycle Architecture", theory))

	return sb.String()
}
