package kernel

import (
	"fmt"
	"os-simulator/pkg/cpu"
	"os-simulator/pkg/ipc"
	"os-simulator/pkg/memory"
	"os-simulator/pkg/visualizer"
	"strings"
	"sync"
)

// OSKernel is the unified operating system kernel integrating CPU, Memory, Processes, Syscalls, Traps, and IPC
type OSKernel struct {
	CPU             *HardwareCPU
	ContextSwitcher *cpu.ContextSwitcher
	Scheduler       cpu.Scheduler
	ProcManager     *ProcessManager
	Syscalls        *SyscallDispatcher
	Traps           *TrapTable
	MemManager      *memory.MemoryManager
	ShmManager      *ipc.SharedMemoryManager
	ActivePipes     map[int]*ipc.UnixPipe
	RunningProcess  *UnixProcess
	CurrentTick     int
	EventHistory    []string
	mu              sync.Mutex
}

func NewOSKernel(sched cpu.Scheduler) *OSKernel {
	if sched == nil {
		sched = cpu.NewRRScheduler(2) // Default Round Robin with quantum 2
	}

	k := &OSKernel{
		CPU:             NewHardwareCPU(),
		ContextSwitcher: cpu.NewContextSwitcher(),
		Scheduler:       sched,
		ProcManager:     NewProcessManager(),
		Syscalls:        NewSyscallDispatcher(),
		Traps:           NewTrapTable(),
		MemManager:      memory.NewMemoryManager(8, memory.PolicyClock, 8),
		ShmManager:      ipc.NewSharedMemoryManager(),
		ActivePipes:     make(map[int]*ipc.UnixPipe),
		EventHistory:    make([]string, 0),
	}

	// Initialize with init process (PID 1) running
	k.RunningProcess = k.ProcManager.Processes[1]
	k.Scheduler.AddProcess(cpu.NewProcess(1, "init", 0, 100))

	return k
}

// Step advances kernel execution by 1 tick (Simulates hardware CPU cycle + Timer Interrupt)
func (k *OSKernel) Step() string {
	k.mu.Lock()
	defer k.mu.Unlock()

	k.CurrentTick++

	// 1. Tick scheduler
	schedProc, _ := k.Scheduler.Tick(k.CurrentTick)

	var logMsg string
	if schedProc != nil {
		if k.RunningProcess == nil || k.RunningProcess.PID != schedProc.PID {
			// Context switch occurred
			oldProcName := "IDLE"
			if k.RunningProcess != nil {
				oldProcName = fmt.Sprintf("P%d(%s)", k.RunningProcess.PID, k.RunningProcess.Name)
			}
			k.RunningProcess = k.ProcManager.Processes[schedProc.PID]
			logMsg = fmt.Sprintf("[Tick %d] Context Switch: %s ──► P%d(%s) | Scheduler: %s",
				k.CurrentTick, oldProcName, schedProc.PID, schedProc.Name, k.Scheduler.Name())
		} else {
			logMsg = fmt.Sprintf("[Tick %d] Executing P%d(%s) in User Mode (PC: 0x%08X)",
				k.CurrentTick, schedProc.PID, schedProc.Name, k.CPU.ProgramCounter)
			k.CPU.ProgramCounter += 4
		}
	} else {
		logMsg = fmt.Sprintf("[Tick %d] CPU Idle (No runnable processes)", k.CurrentTick)
	}

	k.EventHistory = append(k.EventHistory, logMsg)
	return logMsg
}

// TriggerSyscall simulates a user process executing a system call trap
func (k *OSKernel) TriggerSyscall(pid int, syscallNum int, args [6]uint64) (int64, string) {
	k.mu.Lock()
	defer k.mu.Unlock()

	regs := &SyscallRegisters{
		RAX: uint64(syscallNum),
		RDI: args[0],
		RSI: args[1],
		RDX: args[2],
		R10: args[3],
		R8:  args[4],
		R9:  args[5],
	}

	// 1. Enter Kernel Mode via Trap Gate
	k.CPU.SwitchToKernelMode(0x80)

	// 2. Dispatch Syscall
	ret, err := k.Syscalls.Dispatch(pid, regs)

	// 3. Return to User Mode
	k.CPU.ReturnToUserMode()

	status := visualizer.Green("SUCCESS")
	if err != nil {
		status = visualizer.Red(fmt.Sprintf("ERROR: %v", err))
	}

	msg := fmt.Sprintf("[SYSCALL TRAP] PID %d executed Syscall #%d -> Return: %d (%s)",
		pid, syscallNum, ret, status)
	k.EventHistory = append(k.EventHistory, msg)
	return ret, msg
}

// RenderDashboard visualizes the entire live OS state
func (k *OSKernel) RenderDashboard() string {
	k.mu.Lock()
	defer k.mu.Unlock()

	var sb strings.Builder
	sb.WriteString(visualizer.SectionHeader(fmt.Sprintf("UNIFIED OS KERNEL LIVE DASHBOARD (TICK: %d)", k.CurrentTick)))

	// 1. Hardware & CPU State Box
	curPIDStr := "IDLE"
	if k.RunningProcess != nil {
		curPIDStr = fmt.Sprintf("PID %d (%s)", k.RunningProcess.PID, k.RunningProcess.Name)
	}

	cpuState := []string{
		fmt.Sprintf("CPU Privilege Mode:     %s", visualizer.Bold+k.CPU.CurrentMode.String()+visualizer.Reset),
		fmt.Sprintf("Currently Running:      %s%s%s", visualizer.FgHiGreen, curPIDStr, visualizer.Reset),
		fmt.Sprintf("Program Counter (RIP):  0x%08X", k.CPU.ProgramCounter),
		fmt.Sprintf("Stack Pointer (RSP):    0x%08X", k.CPU.UserRSP),
		fmt.Sprintf("Page Table Root (CR3):  0x%08X", k.CPU.CR3),
		fmt.Sprintf("Active CPU Scheduler:   %s", k.Scheduler.Name()),
		fmt.Sprintf("Interrupts Enabled:     %v", k.CPU.InterruptsOn),
	}
	sb.WriteString(visualizer.Box("Hardware CPU & MMU State", cpuState))

	// 2. Process Table
	sb.WriteString("\n" + k.ProcManager.RenderProcessTree())

	// 3. Memory & TLB Metrics
	memStats := []string{
		fmt.Sprintf("Hardware TLB Hit Rate: %.2f%% (%d Hits / %d Misses)", k.MemManager.TLB.HitRate(), k.MemManager.TLB.Hits, k.MemManager.TLB.Misses),
		fmt.Sprintf("Total Page Faults:     %d", k.MemManager.PageFaults),
		fmt.Sprintf("Page Evictions:        %d (Policy: %s)", k.MemManager.PageEvictions, k.MemManager.Policy),
		fmt.Sprintf("Swap Disk Reads/Writes:%d / %d", k.MemManager.SwapReads, k.MemManager.SwapWrites),
	}
	sb.WriteString("\n" + visualizer.Box("Virtual Memory & Paging Subsystem", memStats))

	// 4. Recent Kernel Logs
	if len(k.EventHistory) > 0 {
		start := 0
		if len(k.EventHistory) > 5 {
			start = len(k.EventHistory) - 5
		}
		sb.WriteString("\n" + visualizer.Box("Recent Kernel Event Logs", k.EventHistory[start:]))
	}

	return sb.String()
}
