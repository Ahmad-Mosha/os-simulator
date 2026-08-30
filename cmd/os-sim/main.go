package main

import (
	"bufio"
	"fmt"
	"os"
	"os-simulator/pkg/concurrency"
	"os-simulator/pkg/cpu"
	ioPkg "os-simulator/pkg/io"
	"os-simulator/pkg/ipc"
	"os-simulator/pkg/kernel"
	"os-simulator/pkg/memory"
	"os-simulator/pkg/parallelism"
	"os-simulator/pkg/visualizer"
	"strconv"
	"strings"
	"time"
)

func main() {
	if len(os.Args) > 1 {
		switch strings.ToLower(os.Args[1]) {
		case "cpu":
			runCPULab()
		case "gmp":
			runGMPLab()
		case "memory":
			runMemoryLab()
		case "paging":
			runPagingLab()
		case "concurrency":
			runConcurrencyLab()
		case "deadlock":
			runDeadlockLab()
		case "io":
			runIOLab()
		case "bench":
			runBenchmarkLab()
		case "mode":
			runModeLab()
		case "syscall":
			runSyscallLab()
		case "lifecycle":
			runLifecycleLab()
		case "isolation":
			runIsolationLab()
		case "ipc":
			runIPCLab()
		case "debug":
			runInteractiveDebugger()
		case "all":
			runAllLabs()
		case "help", "--help", "-h":
			printUsage()
		default:
			fmt.Printf("Unknown command: %s\n", os.Args[1])
			printUsage()
		}
		return
	}

	// Interactive Menu
	runInteractiveMenu()
}

func printUsage() {
	fmt.Println(visualizer.SectionHeader("OS SIMULATOR & LAB (OSTEP in Go)"))
	fmt.Println(visualizer.HiCyan("Usage: os-sim [command]"))
	fmt.Println("\nAvailable Labs & Interactive Modules:")
	fmt.Println("  mode         - User Mode vs Kernel Mode, Ring 0/3 Privileges & Traps")
	fmt.Println("  syscall      - System Calls, Dispatch Table & Register Calling Convention (RAX, RDI..)")
	fmt.Println("  lifecycle    - Process Lifecycle (fork, exec, wait, exit, Zombies & Orphans)")
	fmt.Println("  isolation    - Process Memory Isolation & Protection Fault (#PF/SIGSEGV)")
	fmt.Println("  ipc          - Inter-Process Communication (Unix Pipes & Zero-Copy Shared Memory)")
	fmt.Println("  cpu          - CPU Virtualization, Context Switch & Schedulers (FIFO, SJF, STCF, RR, MLFQ, Lottery)")
	fmt.Println("  gmp          - Go M:N GMP Runtime (Goroutines G, Threads M, Processors P, Work Stealing)")
	fmt.Println("  memory       - Memory Virtualization (Address Space, Stack vs Heap, Allocators)")
	fmt.Println("  paging       - Hardware Paging, Multi-Level Page Tables, TLB Cache, Page Replacement (Clock/LRU)")
	fmt.Println("  concurrency  - Data Race Demonstration, Atomics, Mutexes, Semaphores, Condition Variables")
	fmt.Println("  deadlock     - Banker's Algorithm, Coffman Conditions, Dining Philosophers")
	fmt.Println("  io           - I/O Multiplexing (Blocking, Select/Poll, Epoll O(1) Ready-List, Netpoller)")
	fmt.Println("  bench        - Concurrency vs Parallelism Real Hardware Multi-Core Benchmarks")
	fmt.Println("  debug        - Step-by-Step Interactive Unified OS Debugger & Shell")
	fmt.Println("  all          - Run all labs in sequence (Full Guided Tour)")
	fmt.Println("  help         - Show this help message")
}

func runInteractiveMenu() {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Println(visualizer.SectionHeader("OS SIMULATOR & LAB: INTERACTIVE CONSOLE"))
		fmt.Println(visualizer.Bold + "Select an Operating System Lab to run:" + visualizer.Reset)
		fmt.Println(visualizer.HiCyan("── Operating System Kernel & Hardware Boundary ────────────────────────"))
		fmt.Println("  [1] User Mode vs Kernel Mode & Traps (Ring 0 vs Ring 3)")
		fmt.Println("  [2] System Calls Dispatcher & Register Calling Convention")
		fmt.Println("  [3] Unix Process Lifecycle (fork, exec, exit, wait, Zombies & Orphans)")
		fmt.Println("  [4] Process Memory Isolation & Protection Faults")
		fmt.Println("  [5] Inter-Process Communication (Unix Pipes & Shared Memory)")
		fmt.Println("  [6] Unified OS Kernel Step-by-Step Interactive Debugger")
		fmt.Println(visualizer.HiYellow("\n── CPU, Memory & Concurrency Fundamentals (OSTEP) ─────────────────────"))
		fmt.Println("  [7] CPU Scheduling & Context Switching (FIFO, SJF, STCF, RR, MLFQ, Lottery)")
		fmt.Println("  [8] Go Runtime M:N Scheduler (GMP Model, Work Stealing, Stack Dynamics)")
		fmt.Println("  [9] Memory Virtualization (Address Space, Stack vs Heap, Free-List & Buddy)")
		fmt.Println(" [10] Hardware Paging, Multi-Level Page Tables, TLB Cache & Clock Replacement")
		fmt.Println(" [11] Concurrency Primitives, Data Races, Atomics, Mutex, Semaphores, CondVar")
		fmt.Println(" [12] Deadlock Avoidance, Banker's Algorithm & Dining Philosophers")
		fmt.Println(" [13] I/O Models, Epoll O(1) Architecture & Go Netpoller Integration")
		fmt.Println(" [14] Concurrency vs True Parallelism Multi-Core Hardware Benchmarks")
		fmt.Println(" [15] Run Complete Guided Tour (All Labs)")
		fmt.Println("  [0] Exit")
		fmt.Print(visualizer.HiYellow("\nEnter choice (0-15): ") + visualizer.Reset)

		if !scanner.Scan() {
			break
		}
		choice := strings.TrimSpace(scanner.Text())
		switch choice {
		case "1":
			runModeLab()
		case "2":
			runSyscallLab()
		case "3":
			runLifecycleLab()
		case "4":
			runIsolationLab()
		case "5":
			runIPCLab()
		case "6":
			runInteractiveDebugger()
		case "7":
			runCPULab()
		case "8":
			runGMPLab()
		case "9":
			runMemoryLab()
		case "10":
			runPagingLab()
		case "11":
			runConcurrencyLab()
		case "12":
			runDeadlockLab()
		case "13":
			runIOLab()
		case "14":
			runBenchmarkLab()
		case "15":
			runAllLabs()
		case "0", "exit", "quit":
			fmt.Println(visualizer.Green("\nThank you for exploring Operating Systems with Go!"))
			return
		default:
			fmt.Println(visualizer.Red("Invalid selection. Please choose 0-15."))
		}

		fmt.Print(visualizer.Gray("\nPress Enter to return to menu..."))
		scanner.Scan()
	}
}

// ---------------- LAB: User Mode vs Kernel Mode & Traps ----------------
func runModeLab() {
	fmt.Println(visualizer.SectionHeader("LAB: USER MODE (RING 3) vs KERNEL MODE (RING 0)"))
	fmt.Println(kernel.RenderModeComparison())

	cpuHw := kernel.NewHardwareCPU()
	tt := kernel.NewTrapTable()

	fmt.Println(visualizer.SubHeader("Step 1: Attempting to run Privileged HLT in User Mode (Ring 3)..."))
	haltInst := kernel.CPUInstruction{Opcode: kernel.InstPrivilegedHalt, Description: "HLT (Stop CPU)", IsPrivileged: true}
	err := cpuHw.ExecuteInstruction(haltInst)
	if err != nil {
		fmt.Printf("  %s\n", visualizer.Red(fmt.Sprintf("TRAP TRIGGERED: %v", err)))
	}

	fmt.Println(visualizer.SubHeader("Step 2: Switching to Kernel Mode via INT 0x80 Syscall Trap..."))
	cpuHw.SwitchToKernelMode(0x80)
	fmt.Println(visualizer.Green("  CPU is now in KERNEL MODE (Ring 0)"))

	fmt.Println(visualizer.SubHeader("Step 3: Re-executing Privileged Instruction in Kernel Mode..."))
	err = cpuHw.ExecuteInstruction(haltInst)
	if err == nil {
		fmt.Println(visualizer.Green("  Instruction executed successfully in Kernel Mode!"))
	}

	cpuHw.ReturnToUserMode()
	fmt.Println("\n" + tt.RenderIDT())
}

// ---------------- LAB: System Calls & Register Calling Convention ----------------
func runSyscallLab() {
	fmt.Println(visualizer.SectionHeader("LAB: SYSTEM CALL DISPATCHER & CALLING CONVENTIONS"))

	sd := kernel.NewSyscallDispatcher()
	fmt.Println(sd.RenderSyscallTable())

	fmt.Println(visualizer.SubHeader("Executing Step-by-Step Simulated Syscalls"))

	// 1. getpid()
	regs1 := &kernel.SyscallRegisters{RAX: kernel.SysGetPID}
	ret1, _ := sd.Dispatch(101, regs1)
	fmt.Printf("  • getpid() ──► PID: %s%d%s (RAX=%d)\n", visualizer.FgHiGreen, ret1, visualizer.Reset, regs1.RAX)

	// 2. write(fd=1, buf=0x00401000, count=28)
	regs2 := &kernel.SyscallRegisters{
		RAX: kernel.SysWrite,
		RDI: 1,
		RSI: 0x00401000,
		RDX: 28,
	}
	ret2, _ := sd.Dispatch(101, regs2)
	fmt.Printf("  • write(fd=1, buf=0x00401000, count=28) ──► Written: %s%d bytes%s (RAX=%d)\n",
		visualizer.FgHiGreen, ret2, visualizer.Reset, regs2.RAX)

	// 3. write with illegal kernel pointer (0xC0005000)
	regs3 := &kernel.SyscallRegisters{
		RAX: kernel.SysWrite,
		RDI: 1,
		RSI: 0xC0005000, // Invalid pointer in kernel space
		RDX: 28,
	}
	_, err := sd.Dispatch(101, regs3)
	fmt.Printf("  • write(fd=1, buf=0xC0005000) ──► %s\n", visualizer.Red(fmt.Sprintf("REJECTED: %v (RAX=%d)", err, regs3.RAX)))
}

// ---------------- LAB: Unix Process Lifecycle (fork, exec, wait, exit) ----------------
func runLifecycleLab() {
	fmt.Println(visualizer.SectionHeader("LAB: UNIX PROCESS LIFECYCLE (FORK, EXEC, EXIT, WAIT, ZOMBIES)"))

	pm := kernel.NewProcessManager()
	fmt.Println(pm.RenderProcessTree())

	fmt.Println(visualizer.HiCyan("\nStep 1: Init (PID 1) executes fork() to create child process..."))
	child1, _ := pm.Fork(1)
	fmt.Println(pm.RenderProcessTree())

	fmt.Println(visualizer.HiCyan(fmt.Sprintf("\nStep 2: Child (PID %d) executes execve('web_server')...", child1.PID)))
	pm.Exec(child1.PID, "web_server", []string{"web_server", "--port", "8080"})
	fmt.Println(pm.RenderProcessTree())

	fmt.Println(visualizer.HiCyan(fmt.Sprintf("\nStep 3: Web server (PID %d) exits with code 0 ──► Enters ZOMBIE state...", child1.PID)))
	pm.Exit(child1.PID, 0)
	fmt.Println(pm.RenderProcessTree())

	fmt.Println(visualizer.HiCyan("\nStep 4: Parent (PID 1) calls wait() to reap Zombie child..."))
	reapedPID, exitCode, _ := pm.Wait(1)
	fmt.Printf("  Parent reaped child PID %s%d%s with exit status %d (PCB deleted)\n",
		visualizer.FgHiGreen, reapedPID, visualizer.Reset, exitCode)
	fmt.Println(pm.RenderProcessTree())
}

// ---------------- LAB: Process Memory Isolation ----------------
func runIsolationLab() {
	fmt.Println(visualizer.SectionHeader("LAB: PROCESS MEMORY ISOLATION & PROTECTION FAULTS"))

	lab := kernel.NewMemoryIsolationLab()
	res := lab.RunIsolationTest()
	fmt.Println(kernel.RenderIsolationReport(res))
}

// ---------------- LAB: Inter-Process Communication (IPC) ----------------
func runIPCLab() {
	fmt.Println(visualizer.SectionHeader("LAB: INTER-PROCESS COMMUNICATION (PIPES & SHARED MEMORY)"))

	// 1. Unix Pipe
	pipe := ipc.NewUnixPipe(64)
	fmt.Println(visualizer.HiCyan("Writing 26 bytes to in-kernel Pipe buffer..."))
	pipe.Write([]byte("Hello from Process A!"))
	fmt.Println(pipe.RenderPipeStatus())

	buf := make([]byte, 64)
	n, _ := pipe.Read(buf)
	fmt.Printf("  [Process B] Read from pipe: \"%s%s%s\" (%d bytes)\n",
		visualizer.FgHiGreen, string(buf[:n]), visualizer.Reset, n)

	// 2. Shared Memory
	smm := ipc.NewSharedMemoryManager()
	seg, _ := smm.CreateSegment(64)

	pt1 := memory.NewPageTable(10, 1024, 4096)
	smm.Attach(seg.ShmID, 10, 0x00500000, pt1)

	pt2 := memory.NewPageTable(20, 1024, 4096)
	smm.Attach(seg.ShmID, 20, 0x00700000, pt2)

	seg.WriteSync([]byte("High-Speed Zero-Copy Token"))
	fmt.Println(smm.RenderSharedMemoryDiagram(seg.ShmID))
}

// ---------------- LAB: Unified OS Kernel Interactive Step Debugger ----------------
func runInteractiveDebugger() {
	fmt.Println(visualizer.SectionHeader("UNIFIED OS KERNEL: STEP-BY-STEP INTERACTIVE DEBUGGER"))
	k := kernel.NewOSKernel(cpu.NewRRScheduler(2))

	// Pre-populate with some processes
	p2, _ := k.ProcManager.Fork(1)
	k.ProcManager.Exec(p2.PID, "http_server", nil)
	k.Scheduler.AddProcess(cpu.NewProcess(p2.PID, "http_server", 0, 15))

	p3, _ := k.ProcManager.Fork(1)
	k.ProcManager.Exec(p3.PID, "db_worker", nil)
	k.Scheduler.AddProcess(cpu.NewProcess(p3.PID, "db_worker", 0, 10))

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println(k.RenderDashboard())

	fmt.Println(visualizer.HiYellow("Debugger Commands:") + " [s/step] advance tick | [fork] fork child | [exec <name>] exec | [kill <pid>] exit | [wait] reap zombie | [dash] redraw | [q/quit] return")

	for {
		fmt.Print(visualizer.HiCyan("\n[os-kernel-dbg]> ") + visualizer.Reset)
		if !scanner.Scan() {
			break
		}
		cmd := strings.TrimSpace(scanner.Text())
		parts := strings.Fields(cmd)
		if len(parts) == 0 {
			continue
		}

		switch strings.ToLower(parts[0]) {
		case "s", "step":
			stepCount := 1
			if len(parts) > 1 {
				if n, err := strconv.Atoi(parts[1]); err == nil && n > 0 {
					stepCount = n
				}
			}
			for i := 0; i < stepCount; i++ {
				msg := k.Step()
				fmt.Println("  " + msg)
			}

		case "dash", "d":
			fmt.Println(k.RenderDashboard())

		case "fork", "f":
			parentPID := 1
			if k.RunningProcess != nil {
				parentPID = k.RunningProcess.PID
			}
			child, err := k.ProcManager.Fork(parentPID)
			if err != nil {
				fmt.Println(visualizer.Red(fmt.Sprintf("Fork error: %v", err)))
			} else {
				k.Scheduler.AddProcess(cpu.NewProcess(child.PID, child.Name, k.CurrentTick, 8))
				fmt.Println(visualizer.Green(fmt.Sprintf("Forked child PID %d under parent PID %d", child.PID, parentPID)))
			}

		case "exec", "e":
			if len(parts) < 2 {
				fmt.Println(visualizer.Red("Usage: exec <binary_name>"))
				continue
			}
			if k.RunningProcess == nil {
				fmt.Println(visualizer.Red("No running process to exec"))
				continue
			}
			k.ProcManager.Exec(k.RunningProcess.PID, parts[1], nil)
			fmt.Println(visualizer.Green(fmt.Sprintf("PID %d executed new binary '%s'", k.RunningProcess.PID, parts[1])))

		case "kill", "exit":
			targetPID := 1
			if len(parts) > 1 {
				targetPID, _ = strconv.Atoi(parts[1])
			} else if k.RunningProcess != nil {
				targetPID = k.RunningProcess.PID
			}
			err := k.ProcManager.Exit(targetPID, 0)
			if err != nil {
				fmt.Println(visualizer.Red(fmt.Sprintf("Exit error: %v", err)))
			} else {
				fmt.Println(visualizer.Yellow(fmt.Sprintf("PID %d exited -> ZOMBIE", targetPID)))
			}

		case "wait", "w":
			parentPID := 1
			if len(parts) > 1 {
				parentPID, _ = strconv.Atoi(parts[1])
			}
			reaped, exitCode, err := k.ProcManager.Wait(parentPID)
			if err != nil {
				fmt.Println(visualizer.Red(fmt.Sprintf("Wait error: %v", err)))
			} else {
				fmt.Println(visualizer.Green(fmt.Sprintf("Parent PID %d reaped Zombie child PID %d (Exit Code: %d)", parentPID, reaped, exitCode)))
			}

		case "ps":
			fmt.Println(k.ProcManager.RenderProcessTree())

		case "q", "quit", "exit-dbg":
			return

		case "help", "h":
			fmt.Println("Commands: step [N], dash, fork, exec <name>, kill <pid>, wait [parentPID], ps, quit")

		default:
			fmt.Println("Unknown command. Type 'help' for options.")
		}
	}
}

// ---------------- LAB 1: CPU Scheduling & Context Switching ----------------
func runCPULab() {
	fmt.Println(visualizer.SectionHeader("LAB 1: CPU VIRTUALIZATION & SCHEDULING"))

	procs := []*cpu.Process{
		cpu.NewProcess(1, "Job_A", 0, 8),
		cpu.NewProcess(2, "Job_B", 1, 4),
		cpu.NewProcess(3, "Job_C", 3, 2),
	}

	schedulers := []cpu.Scheduler{
		cpu.NewFIFOScheduler(),
		cpu.NewSJFScheduler(),
		cpu.NewSTCFScheduler(),
		cpu.NewRRScheduler(2),
		cpu.NewMLFQScheduler(30),
		cpu.NewLotteryScheduler(2, 42),
	}

	for _, s := range schedulers {
		fmt.Println(visualizer.SubHeader(fmt.Sprintf("Running %s", s.Name())))
		for _, p := range procs {
			s.AddProcess(p.Clone())
		}

		for t := 0; t < 100; t++ {
			_, done := s.Tick(t)
			if done {
				break
			}
		}

		fmt.Println("Gantt Timeline Chart:")
		fmt.Print(visualizer.RenderGanttChart(s.GetGantt()))
		fmt.Println(cpu.FormatMetricsTable(s.Name(), s.GetMetrics(), s.GetCompleted()))
	}

	cs := cpu.NewContextSwitcher()
	rec := cs.SwitchProcessContext(10, procs[0], procs[1])
	fmt.Println(cs.ExplainContextSwitch(rec))
}

// ---------------- LAB 2: Go Runtime M:N Scheduler (GMP) ----------------
func runGMPLab() {
	fmt.Println(visualizer.SectionHeader("LAB 2: GO RUNTIME M:N SCHEDULER (GMP MODEL)"))

	rt := cpu.NewGMPRuntime(2)

	fmt.Println(visualizer.HiCyan("Spawning Goroutines (G1..G5 CPU tasks + G6 Blocking Syscall)..."))
	for i := 1; i <= 5; i++ {
		rt.SpawnGoroutine(fmt.Sprintf("worker_%d", i), 4, false)
	}
	rt.SpawnGoroutine("disk_read_syscall", 6, true)

	for tick := 0; tick < 15; tick++ {
		rt.Step(tick)
	}

	fmt.Println(rt.RenderStatus())
}

// ---------------- LAB 3: Memory Virtualization (Address Space, Stack & Heap) ----------------
func runMemoryLab() {
	fmt.Println(visualizer.SectionHeader("LAB 3: MEMORY VIRTUALIZATION & ALLOCATORS"))

	as := memory.NewAddressSpace(101, 64*1024, 0x00200000)
	fmt.Println(as.RenderMemoryMap())

	cs := memory.NewCallStack(0xFFFF, 16*1024)
	cs.PushFrame("main", 0x00400000, 64, map[string]interface{}{"argc": 1, "status": 0})
	cs.PushFrame("processData", 0x00401200, 48, map[string]interface{}{"bufferSize": 1024, "count": 5})
	cs.PushFrame("calculateHash", 0x00402400, 32, map[string]interface{}{"seed": 0xDEADBEEF})

	fmt.Println(cs.RenderStack())

	fla := memory.NewFreeListAllocator(0x00010000, 4096)
	b1, _ := fla.Malloc(512, false)
	b2, _ := fla.Malloc(1024, false)
	b3, _ := fla.Malloc(256, false)
	_ = b3

	fmt.Println(visualizer.SubHeader("Allocated 3 Heap Blocks (b1:512B, b2:1024B, b3:256B)"))
	fmt.Println(fla.RenderHeapMap())

	fmt.Println(visualizer.HiYellow("\nFreeing middle block (b2) and first block (b1) to demonstrate Coalescing..."))
	fla.Free(b2)
	fla.Free(b1)
	fmt.Println(fla.RenderHeapMap())
}

// ---------------- LAB 4: Hardware Paging, Multi-Level PT, TLB & Clock ----------------
func runPagingLab() {
	fmt.Println(visualizer.SectionHeader("LAB 4: HARDWARE PAGING, MULTI-LEVEL PT, TLB & CLOCK"))

	pt := memory.NewPageTable(1, 16, 4096)
	pt.MapPage(0, 10, true)
	pt.MapPage(1, 14, false)
	pt.MapPage(2, 22, true)
	fmt.Println(pt.RenderPageTable())

	mlpt := memory.NewMultiLevelPageTable(1)
	mlpt.Map(0x00400000, 50, true)
	mlpt.Map(0x7FFF0000, 80, true)
	fmt.Println(mlpt.RenderTwoLevelStatus())

	tlb := memory.NewTLB(4)
	tlb.Insert(1, 0, 10, true)
	tlb.Insert(1, 1, 14, false)
	tlb.Lookup(1, 0)
	tlb.Lookup(1, 2)
	tlb.Lookup(1, 1)
	fmt.Println(tlb.RenderTLBStatus())

	mm := memory.NewMemoryManager(3, memory.PolicyClock, 4)
	for i := 0; i < 6; i++ {
		pt.MapPage(uint64(i), 0, true)
		pt.Entries[i].Present = false
	}

	fmt.Println(visualizer.HiCyan("Simulating page accesses: 0, 1, 2, 3, 0, 4 (Frame capacity = 3)..."))
	accessPattern := []uint64{0, 1, 2, 3, 0, 4}
	for _, page := range accessPattern {
		mm.AccessMemory(pt, page*4096, page%2 == 1)
	}
	fmt.Println(mm.RenderPhysicalMemory())
}

// ---------------- LAB 5: Concurrency Primitives & Data Races ----------------
func runConcurrencyLab() {
	fmt.Println(visualizer.SectionHeader("LAB 5: CONCURRENCY, DATA RACES & SYNCHRONIZATION"))

	fmt.Println(visualizer.HiCyan("Executing live parallel Data Race test (10 Goroutines x 10,000 iterations)..."))
	res := concurrency.RunDataRaceDemo(10, 10000)
	fmt.Println(concurrency.RenderRaceExplanation(res))

	fmt.Println(visualizer.SubHeader("Producer-Consumer with Mesa-Semantics Condition Variable"))
	bb := concurrency.NewBoundedBuffer(3)
	go func() {
		for i := 1; i <= 5; i++ {
			bb.Put(i * 10)
			time.Sleep(5 * time.Millisecond)
		}
	}()

	for i := 1; i <= 5; i++ {
		val := bb.Get()
		fmt.Printf("  [Consumer] Extracted item: %s%d%s from bounded buffer\n", visualizer.FgHiGreen, val, visualizer.Reset)
	}
}

// ---------------- LAB 6: Deadlock Avoidance & Banker's Algorithm ----------------
func runDeadlockLab() {
	fmt.Println(visualizer.SectionHeader("LAB 6: DEADLOCK AVOIDANCE & DINING PHILOSOPHERS"))

	resNames := []string{"CPU", "Disk", "Memory"}
	total := []int{10, 5, 7}
	b := concurrency.NewBankersState(5, resNames, total)

	b.SetProcessClaim(0, []int{7, 5, 3}, []int{0, 1, 0})
	b.SetProcessClaim(1, []int{3, 2, 2}, []int{2, 0, 0})
	b.SetProcessClaim(2, []int{9, 0, 2}, []int{3, 0, 2})
	b.SetProcessClaim(3, []int{2, 2, 2}, []int{2, 1, 1})
	b.SetProcessClaim(4, []int{4, 3, 3}, []int{0, 0, 2})

	fmt.Println(b.RenderState())
	isSafe, seq, log := b.CheckSafeState()
	fmt.Printf("Safe State: %v\nSequence: P%v\n%s\n", isSafe, seq, log)

	dp := concurrency.NewDiningPhilosophers(5)
	dp.RunSafeResourceHierarchy(5)
	fmt.Println(dp.RenderDiningStatus())
}

// ---------------- LAB 7: I/O Multiplexing & Go Netpoller ----------------
func runIOLab() {
	fmt.Println(visualizer.SectionHeader("LAB 7: I/O MULTIPLEXING, EPOLL & GO NETPOLLER"))

	fmt.Println(ioPkg.RenderEpollComparison(10000, 2))

	np := ioPkg.NewGoNetpoller()
	fmt.Println(np.RenderNetpollerExplanation())
}

// ---------------- LAB 8: Concurrency vs True Parallelism ----------------
func runBenchmarkLab() {
	fmt.Println(visualizer.SectionHeader("LAB 8: CONCURRENCY vs TRUE PARALLELISM BENCHMARK"))

	fmt.Println(visualizer.HiCyan("Running live hardware benchmarks across single core vs multi-core..."))
	benchRes := parallelism.RunComparisonBenchmark(8, 200000)
	fmt.Println(parallelism.RenderBenchmarkReport(benchRes))
}

func runAllLabs() {
	runModeLab()
	runSyscallLab()
	runLifecycleLab()
	runIsolationLab()
	runIPCLab()
	runCPULab()
	runGMPLab()
	runMemoryLab()
	runPagingLab()
	runConcurrencyLab()
	runDeadlockLab()
	runIOLab()
	runBenchmarkLab()
}
