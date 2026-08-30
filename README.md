# Operating System Simulator & Educational Lab (`os-simulator`)

A comprehensive, production-grade Operating System simulator and interactive laboratory built from scratch in Go. This project bridges theoretical operating system fundamentals—based on the renowned textbook **Operating Systems: Three Easy Pieces (OSTEP)**—with real-world systems programming, hardware-kernel boundaries, and **Go Runtime internals** (M:N scheduler, GMP model, netpoller, dynamic stack allocation, and memory management).

---

## Table of Contents

1. [Architecture & Project Overview](#architecture--project-overview)
2. [Quick Start & Interactive CLI](#quick-start--interactive-cli)
3. [Deep Dive: Kernel & Hardware Boundary](#deep-dive-kernel--hardware-boundary)
   - [User Mode (Ring 3) vs Kernel Mode (Ring 0)](#user-mode-ring-3-vs-kernel-mode-ring-0)
   - [System Calls & Calling Conventions (x86-64 ABI)](#system-calls--calling-conventions-x86-64-abi)
   - [Traps, CPU Exceptions & Interrupt Descriptor Table (IDT)](#traps-cpu-exceptions--interrupt-descriptor-table-idt)
   - [Unix Process Lifecycle: fork, exec, wait, exit, Zombies & Orphans](#unix-process-lifecycle-fork-exec-wait-exit-zombies--orphans)
   - [Process Memory Isolation & Protection Faults](#process-memory-isolation--protection-faults)
   - [Inter-Process Communication (IPC: Pipes, Shared Memory, Message Queues)](#inter-process-communication-ipc)
4. [Deep Dive: CPU Virtualization](#deep-dive-cpu-virtualization)
   - [Process (PCB) vs Thread (TCB) vs Goroutine (G)](#process-pcb-vs-thread-tcb-vs-goroutine-g)
   - [Why Threads and Goroutines Have Private Stacks](#why-threads-and-goroutines-have-private-stacks)
   - [Context Switching Mechanics & Hardware Costs](#context-switching-mechanics--hardware-costs)
   - [CPU Scheduling Algorithms (FIFO, SJF, STCF, RR, MLFQ, Lottery)](#cpu-scheduling-algorithms)
   - [The Go Runtime M:N Scheduler (GMP Model & Work Stealing)](#the-go-runtime-mn-scheduler-gmp-model)
5. [Deep Dive: Memory Virtualization](#deep-dive-memory-virtualization)
   - [Virtual Address Space Layout](#virtual-address-space-layout)
   - [Stack vs Heap Internals](#stack-vs-heap-internals)
   - [Dynamic Memory Allocators (Free-List & Buddy Allocator)](#dynamic-memory-allocators)
   - [Hardware Paging & Address Translation (VPN ➔ PFN)](#hardware-paging--address-translation)
   - [Multi-Level Page Tables & Sparse Address Spaces](#multi-level-page-tables)
   - [Translation Lookaside Buffer (TLB Cache)](#translation-lookaside-buffer-tlb-cache)
   - [Page Faults, Swap Space & Replacement Policies (Clock/LRU)](#page-faults-swap-space--replacement-policies)
6. [Deep Dive: Concurrency & Synchronization](#deep-dive-concurrency--synchronization)
   - [Data Races & Hardware Interleaving (Assembly Trace)](#data-races--hardware-interleaving)
   - [Hardware Atomics (CAS, Test-and-Set, Fetch-and-Add)](#hardware-atomics)
   - [Spinlocks vs Blocking Mutexes (Futex Mechanics)](#spinlocks-vs-blocking-mutexes)
   - [Condition Variables & Mesa Semantics](#condition-variables--mesa-semantics)
   - [Semaphores & Reader-Writer Locks](#semaphores--reader-writer-locks)
   - [Deadlocks, Coffman Conditions & Banker's Algorithm](#deadlocks-coffman-conditions--bankers-algorithm)
   - [Dining Philosophers Problem & Resource Hierarchy Proof](#dining-philosophers-problem)
7. [Deep Dive: I/O Models & Event Multiplexing](#deep-dive-io-models--event-multiplexing)
   - [Blocking I/O vs Non-Blocking I/O](#blocking-io-vs-non-blocking-io)
   - [Select/Poll $O(N)$ Bottleneck vs Epoll $O(1)$ Architecture](#selectpoll-on-bottleneck-vs-epoll-o1-architecture)
   - [Go Runtime Netpoller Integration](#go-runtime-netpoller-integration)
8. [Concurrency vs True Parallelism](#concurrency-vs-true-parallelism)
9. [Step-by-Step Interactive Debugger](#step-by-step-interactive-debugger)
10. [Codebase & Package Reference Guide](#codebase--package-reference-guide)

---

## Architecture & Project Overview

```
os-simulator/
├── go.mod
├── .gitignore
├── cmd/
│   └── os-sim/
│       └── main.go              # Interactive CLI / TUI with ANSI visuals & lab runners
├── pkg/
│   ├── kernel/                  # Kernel Core & Hardware-OS Boundary
│   │   ├── cpu_mode.go          # User Mode (Ring 3) vs Kernel Mode (Ring 0), privilege traps
│   │   ├── trap_table.go        # IDT / Trap Vector Table, Exceptions, Hardware Interrupts & ISRs
│   │   ├── syscall.go           # Syscall Dispatcher, calling convention (RAX, RDI..), validation
│   │   ├── process_lifecycle.go # fork(), exec(), exit(), wait(), Zombie & Orphan state machine
│   │   ├── isolation.go         # Virtual address space isolation & hardware memory protection faults
│   │   └── os_kernel.go         # Unified OS Kernel integrating CPU, Memory, Threads, Locks & IPC
│   ├── ipc/                     # Inter-Process Communication
│   │   ├── pipe.go              # Unix Pipes: circular ring buffer, blocking read/write, EOF, SIGPIPE
│   │   ├── shared_memory.go     # Shared Physical Frames (zero-copy), multiple VA mappings & IPC semaphores
│   │   └── message_queue.go     # Message Queues: typed priority message passing
│   ├── cpu/                     # CPU Virtualization, Schedulers & Go GMP Runtime
│   │   ├── process.go           # Process Control Block (PCB), lifecycle states, hardware registers
│   │   ├── thread.go            # Thread Control Block (TCB) & private stack boundaries
│   │   ├── context_switch.go    # Register save/restore, cycle accounting & TLB invalidation
│   │   ├── scheduler.go         # Common Scheduler interface & benchmark metrics
│   │   ├── sched_fifo.go        # FIFO / FCFS Non-preemptive scheduler
│   │   ├── sched_sjf.go         # SJF (Non-preemptive) & STCF (Preemptive) schedulers
│   │   ├── sched_rr.go          # Round Robin scheduler with configurable time slices
│   │   ├── sched_mlfq.go        # Multi-Level Feedback Queue with the 5 OSTEP rules
│   │   ├── sched_lottery.go     # Proportional-Share Lottery scheduler with tickets
│   │   └── gmp_runtime.go       # Go Runtime M:N Scheduler (G, M, P, Work Stealing, Handoff)
│   ├── memory/                  # Memory Virtualization, Paging & Allocators
│   │   ├── address_space.go     # Virtual address space layout & Base/Bounds translation
│   │   ├── stack_heap.go        # Call stack activation frames, RSP/RBP & stack overflow
│   │   ├── allocator.go         # Free-List (First/Best-Fit, coalescing) & Buddy Allocator
│   │   ├── paging.go            # Linear Page Table, PTE bits (Valid, Present, Dirty, Accessed)
│   │   ├── multilevel_pt.go     # Two-Level Hierarchical Page Table (PDE + PTE)
│   │   ├── tlb.go               # Hardware TLB cache with Hit/Miss & LRU eviction
│   │   └── page_replacement.go # Page fault handling, Swap space, FIFO, LRU, Clock (Second Chance)
│   ├── concurrency/             # Concurrency Primitives, Atomics & Deadlocks
│   │   ├── race_demo.go         # Live Data Race demo & assembly-level load-modify-store trace
│   │   ├── atomics.go           # Hardware Atomics (CAS, XCHG, LOCK XADD)
│   │   ├── spinlock.go          # Busy-waiting Spinlock
│   │   ├── mutex.go             # Blocking Mutex with Futex-like sleep/wakeup wait queues
│   │   ├── condvar.go           # Condition Variable with Mesa semantics & Bounded Buffer
│   │   ├── semaphore.go         # Dijkstra's Counting Semaphore (P/Wait, V/Signal)
│   │   ├── rwlock.go            # Read-Write Lock with Writer-Preference
│   │   ├── deadlock_banker.go   # Coffman Conditions & Dijkstra's Banker's Algorithm
│   │   └── dining_philosophers.go # Dining Philosophers deadlock vs Resource Hierarchy
│   ├── io/                      # I/O Models & Multiplexing
│   │   ├── blocking_io.go       # Synchronous blocking I/O simulation
│   │   ├── select_poll.go       # O(N) Select/Poll descriptor scanning
│   │   ├── epoll.go             # O(1) Epoll kernel ready-list simulation
│   │   └── netpoller.go         # Go Netpoller: non-blocking sockets + parked Goroutines
│   ├── parallelism/             # Concurrency vs Parallelism
│   │   └── comparison.go        # Multi-core hardware benchmarks & lock contention (Amdahl's Law)
│   └── visualizer/              # Visualization Engine
│       ├── ansi.go              # Colors, box drawing, section banners
│       ├── gantt.go             # ASCII Gantt timeline charts
│       ├── memory_map.go        # ASCII memory layout & translation diagrams
│       └── table.go             # Formatted ASCII tables
```

---

## Quick Start & Interactive CLI

### Build the Project
```bash
go build -o os-sim ./cmd/os-sim
```

### Run Full Test Suite
```bash
go test -v ./pkg/...
```

### Interactive Menu
Run `./os-sim` without arguments to launch the interactive console:
```
══════════════════════════════════════════════════════════════════════
║               OS SIMULATOR & LAB: INTERACTIVE CONSOLE              ║
══════════════════════════════════════════════════════════════════════

Select an Operating System Lab to run:
── Operating System Kernel & Hardware Boundary ────────────────────────
  [1] User Mode vs Kernel Mode & Traps (Ring 0 vs Ring 3)
  [2] System Calls Dispatcher & Register Calling Convention
  [3] Unix Process Lifecycle (fork, exec, exit, wait, Zombies & Orphans)
  [4] Process Memory Isolation & Protection Faults
  [5] Inter-Process Communication (Unix Pipes & Shared Memory)
  [6] Unified OS Kernel Step-by-Step Interactive Debugger

── CPU, Memory & Concurrency Fundamentals (OSTEP) ─────────────────────
  [7] CPU Scheduling & Context Switching (FIFO, SJF, STCF, RR, MLFQ, Lottery)
  [8] Go Runtime M:N Scheduler (GMP Model, Work Stealing, Stack Dynamics)
  [9] Memory Virtualization (Address Space, Stack vs Heap, Free-List & Buddy)
 [10] Hardware Paging, Multi-Level Page Tables, TLB Cache & Clock Replacement
 [11] Concurrency Primitives, Data Races, Atomics, Mutex, Semaphores, CondVar
 [12] Deadlock Avoidance, Banker's Algorithm & Dining Philosophers
 [13] I/O Models, Epoll O(1) Architecture & Go Netpoller Integration
 [14] Concurrency vs True Parallelism Multi-Core Hardware Benchmarks
 [15] Run Complete Guided Tour (All Labs)
  [0] Exit
```

---

## Deep Dive: Kernel & Hardware Boundary

### User Mode (Ring 3) vs Kernel Mode (Ring 0)

| Dimension | User Mode (Ring 3) | Kernel Mode (Ring 0 / Supervisor) |
| :--- | :--- | :--- |
| **Privilege Level** | Lowest (Restricted) | Highest (Full Supervisor) |
| **Memory Access** | User Virtual Address Space segments only | All physical & virtual memory directly |
| **Privileged Instructions** | Blocked (Triggers `#GP` General Protection Fault) | Allowed (`HLT`, `CLI`, `STI`, `MOV CR3`, `IN/OUT`) |
| **I/O Device Access** | Restricted (Must request via Syscall) | Direct device bus access (Ports, MMIO) |
| **Stack Used** | User Space Stack | Private Kernel Stack (Safe from user corruption) |

#### Why Dual-Mode CPU Operation is Essential:
1. **Fault Isolation**: A crashing user program (e.g. division by zero, segfault) cannot crash other programs or bring down the operating system.
2. **Security & Access Control**: Prevents user processes from reading passwords out of raw physical RAM or altering hardware registers directly.
3. **Controlled Gateways**: User code enters Kernel Mode *only* through controlled gates: **Traps**, **System Calls**, and **Hardware Interrupts**.

---

### System Calls & Calling Conventions (x86-64 ABI)

A System Call is the programmatic interface by which user-space software requests services from the operating system kernel.

```
User Program (Ring 3)                    Kernel Mode (Ring 0)
─────────────────────                    ────────────────────
1. Load RAX = 6 (sys_write)
2. Load RDI = 1 (stdout)
3. Load RSI = 0x00401000 (buf)
4. Load RDX = 28 (count)
5. Execute: SYSCALL / INT 0x80  ─────►   6. CPU switches to Ring 0 & Kernel Stack
                                         7. Validate user pointers (< 0xC0000000)
                                         8. Lookup SyscallTable[RAX]
                                         9. Execute sys_write handler
12. Read return value from RAX  ◄─────  10. Set RAX = 28 (bytes written)
                                        11. Execute: SYSRET / IRET
```

#### Calling Convention Register Mapping:
- `RAX`: System call number (input) / Return code or `-errno` (output)
- `RDI`: 1st argument (e.g. `int fd`)
- `RSI`: 2nd argument (e.g. `const void *buf`)
- `RDX`: 3rd argument (e.g. `size_t count`)
- `R10`, `R8`, `R9`: 4th, 5th, 6th arguments

#### Pointer Sanitization in Kernel Mode:
User programs could pass malicious pointers into kernel space (e.g. `0xC0000000+`). The kernel validates all user pointers before dereferencing; any invalid pointer immediately returns `-EFAULT` to protect kernel memory.

---

### Traps, CPU Exceptions & Interrupt Descriptor Table (IDT)

The CPU uses the **Interrupt Descriptor Table (IDT)** to route hardware interrupts, CPU exceptions, and software traps to the appropriate **Interrupt Service Routine (ISR)**:

```
┌────────┬────────────────────────────────┬─────────────────┬──────────────────────────────────────────┐
│ Vector │ Name                           │ Classification  │ Description                              │
├────────┼────────────────────────────────┼─────────────────┼──────────────────────────────────────────┤
│ 0x00   │ Divide-by-Zero (#DE)           │ CPU Exception   │ Integer division by zero                 │
│ 0x06   │ Invalid Opcode (#UD)           │ CPU Exception   │ CPU encountered unknown instruction      │
│ 0x0D   │ General Protection Fault (#GP) │ CPU Exception   │ Privilege violation (e.g. HLT in Ring 3) │
│ 0x0E   │ Page Fault (#PF)               │ CPU Exception   │ Page not present in RAM / bad access     │
│ 0x20   │ PIT Timer Interrupt (IRQ 0)    │ Hardware IRQ    │ Periodic timer for CPU preemption        │
│ 0x21   │ Keyboard Controller (IRQ 1)    │ Hardware IRQ    │ Key press / release event                │
│ 0x80   │ System Call Gate (INT 0x80)    │ Software Trap   │ User space requesting kernel service     │
└────────┴────────────────────────────────┴─────────────────┴──────────────────────────────────────────┘
```

---

### Unix Process Lifecycle: `fork`, `exec`, `wait`, `exit`, Zombies & Orphans

```
                  ┌──────────────┐
                  │   fork()     │  Duplicates PCB & Address Space
                  └──────┬───────┘
                         │
          ┌──────────────┴──────────────┐
          ▼                             ▼
   Parent Process                 Child Process
 (Receives Child PID)           (Receives RAX = 0)
          │                             │
          │                       ┌─────▼────────┐
          │                       │   execve()   │  Replaces code image & stack
          │                       └─────┬────────┘
          │                             │
          │                       ┌─────▼────────┐
          │                       │    exit()    │  Releases memory, stores exit code
          │                       └─────┬────────┘
          │                             │
          │                             ▼
          │                       ┌──────────────┐
          │                       │ ZOMBIE STATE │  (Holds exit status in PCB)
          │                       └──────┬───────┘
          │                              │
          │    ┌─────────────────────────┘
          ▼    ▼
    ┌──────────────┐
    │    wait()    │  Parent reads child exit status & REAPS zombie PCB
    └──────────────┘
```

- **`fork()`**: Clones the calling process. The child receives an exact duplicate of the virtual address space and file descriptors.
- **`execve(path, argv)`**: Replaces the process memory layout with a new executable program, preserving PID, PPID, and open file descriptors.
- **`exit(code)`**: Frees process virtual memory. The process enters the **`ZOMBIE`** state, preserving its exit code in its PCB until the parent calls `wait()`.
- **`wait()` / `waitpid()`**: Blocks the parent until the child terminates, reads the exit code, and **reaps** (destroys) the child PCB.
- **Orphan Processes**: If a parent process dies before its children, the **`init` process (PID 1)** automatically adopts the orphaned children and reaps them when they exit.

---

### Process Memory Isolation & Protection Faults

Virtual memory guarantees that every process operates in a private, isolated address space:

```
Process A (PID 100)                     Physical RAM (Hardware)
Virtual Address: 0x00401050 ──► VPN 0x401 ──► PFN 12 (0x0000C050) ["Secret_Token_A"]

Process B (PID 200)
Virtual Address: 0x00401050 ──► VPN 0x401 ──► PFN 88 (0x00058050) ["Public_Data_B"]
```

- **Zero Collisions**: Even though Process A and Process B use the exact same virtual address `0x00401050`, the MMU translates them to completely different physical frames.
- **Hardware Protection**: If Process B attempts to access physical frames outside its Page Table, the MMU triggers an immediate **Page Fault Exception (`#PF` / `SIGSEGV`)**.

---

### Inter-Process Communication (IPC)

#### 1. Unix Anonymous Pipes
- Unidirectional in-kernel circular ring buffer.
- `pipe(int fds[2])`: `fds[0]` (read end), `fds[1]` (write end).
- Automatic blocking synchronization: Reading from an empty pipe blocks; writing to a full pipe blocks (flow control / backpressure).
- Closing the write end delivers `EOF` (0 bytes) to the reader. Writing to a closed read end raises `SIGPIPE`.

#### 2. Shared Memory (`shmget`, `shmat`)
- The kernel maps the **exact same physical frame (PFN)** into multiple processes' Page Tables.
- **Zero-Copy Performance**: Processes read/write directly to physical RAM at hardware bus speed without memory copies or syscall overhead!
- **Synchronization**: Because access bypasses the kernel, processes must use **IPC Semaphores** to avoid data races.

#### 3. Message Queues
- Kernel-managed linked lists of typed structured messages (`msgsnd`, `msgrcv`).

---

## Deep Dive: CPU Virtualization

### Process (PCB) vs Thread (TCB) vs Goroutine (G)

| Dimension | Process (`Process`) | OS Thread (`Thread`) | Goroutine (`Goroutine`) |
| :--- | :--- | :--- | :--- |
| **Managed By** | OS Kernel | OS Kernel (`pthread`) | Go Runtime (User-Space) |
| **Address Space** | Isolated Virtual Memory (Page Directory) | Shared with parent process | Shared with parent Go process |
| **Control Block** | PCB (Process Control Block) | TCB (Thread Control Block) | `runtime.g` struct |
| **Initial Stack Size** | 2MB – 8MB (Virtual memory) | 2MB – 8MB (Fixed OS stack) | **2 KB** (Dynamically grows/shrinks) |
| **Creation Cost** | Heavy (`fork` + `exec` ~1-5ms) | Medium (`clone`/`pthread_create` ~50µs) | Ultra-light (`go func()` ~100ns) |
| **Switch Cost** | ~2–5 microseconds (TLB flush, cache miss) | ~0.5–1 microsecond (Kernel trap) | **~10–20 nanoseconds** (User-space swap of 3 registers) |
| **Scalability Limit** | Thousands | Thousands (~10,000 max before OOM) | **Millions** |

### Why Threads and Goroutines Have Private Stacks

A process is a **resource container**: it owns the Virtual Address Space (Code, Data, Heap), File Descriptors, and Environment.

A thread or goroutine is an **independent stream of instruction execution**. Because functions can be invoked concurrently in different execution flows:
1. Every thread **must** maintain its own call stack to track its active function activation frames, return addresses, and local variables.
2. If threads shared a single stack, calling a function in Thread A would overwrite the return address and local variables of Thread B, causing instant memory corruption and crashes!
3. However, all threads in a process share the **Heap** and **Data Segment**, allowing fast inter-thread communication (which requires synchronization).

### Context Switching Mechanics & Hardware Costs

During a context switch, the OS / Runtime performs:
1. **Interrupt / System Call**: Hardware switches CPU from User Mode (Ring 3) to Kernel Mode (Ring 0).
2. **Context Save**: Current CPU registers (`PC`, `SP`, `BP`, `AX`, `BX`, `CX`, `DX`, `FLAGS`) are saved into the outgoing PCB / TCB kernel stack.
3. **Scheduler Decision**: Scheduling algorithm chooses the next entity to execute.
4. **Memory Map Switch (Process Switch Only)**: The CPU memory root register (`CR3` on x86, `TTBR0` on ARM) is updated to point to the incoming process's Page Directory.
5. **Hardware Cache Impact**: Switching page tables invalidates the **TLB (Translation Lookaside Buffer)** and causes hardware CPU cache misses ($L1/L2/L3$).
6. **Context Restore**: Incoming registers are restored into hardware registers, and `iret` returns execution to User Mode (Ring 3).

### CPU Scheduling Algorithms

Our simulator implements:

1. **FIFO / FCFS (First-In, First-Out)**:
   - Non-preemptive. Simple FIFO queue.
   - *Problem*: Vulnerable to the **Convoy Effect** (short jobs stuck waiting behind long CPU-bound jobs).
2. **SJF (Shortest Job First)**:
   - Non-preemptive. Minimizes average waiting time if all jobs arrive at $T=0$.
   - *Problem*: Cannot preempt long jobs that arrive slightly earlier.
3. **STCF (Shortest Time-to-Completion First / Preemptive SJF)**:
   - Preempts currently running job whenever a new job arrives with shorter remaining execution time.
   - Optimizes Turnaround Time ($Turnaround = Finish - Arrival$).
4. **Round Robin (RR)**:
   - Preemptive with time slice $Q$ (Quantum).
   - Optimizes Response Time ($Response = FirstRun - Arrival$).
   - Trade-off: Small quantum gives great interactive response but increases context switch overhead.
5. **MLFQ (Multi-Level Feedback Queue)**:
   - Implements the **5 OSTEP Rules**:
     - **Rule 1**: If $Priority(A) > Priority(B)$, $A$ runs ($B$ does not).
     - **Rule 2**: If $Priority(A) == Priority(B)$, $A$ and $B$ run in Round-Robin using the quantum of that queue level.
     - **Rule 3**: When a job enters the system, it enters at the highest priority (Q0).
     - **Rule 4**: Once a job uses up its time allotment at a given level (regardless of how many times it yields for I/O), its priority is reduced (moves down one queue).
     - **Rule 5**: After period $S$ (Boost Period), move ALL jobs to Q0 (**Priority Boost** to prevent starvation and adapt to changing job behavior).
6. **Lottery Scheduling (Proportional Share)**:
   - Assigns lottery tickets to processes.
   - On every quantum, winning ticket is chosen randomly: $P(win) = \frac{Tickets(P)}{\sum Tickets}$.
   - Guarantees probabilistic proportional share of CPU without complex state tracking.

---

### The Go Runtime M:N Scheduler (GMP Model)

Go uses an **M:N hybrid scheduler** where $M$ user-level Goroutines ($G$) are multiplexed onto $N$ kernel OS threads ($M$) across $P$ Logical Processors:

```
                  ┌────────────────────────┐
                  │ Global Run Queue (GRQ) │  (Checked every 61 ticks)
                  └───────────┬────────────┘
                              │
             ┌────────────────┴────────────────┐
             ▼                                 ▼
      ┌─────────────┐                   ┌─────────────┐
      │     P0      │                   │     P1      │   (Logical Processors = GOMAXPROCS)
      │ [Local LRQ] │◄───Work Steal────►│ [Local LRQ] │
      └──────┬──────┘   (Steals 50%)    └──────┬──────┘
             │                                 │
      ┌──────▼──────┐                   ┌──────▼──────┐
      │     M0      │                   │     M1      │   (OS Kernel Threads)
      │  Running G1 │                   │  Running G2 │
      └─────────────┘                   └─────────────┘
```

- **G (Goroutine)**: Contains stack, instruction pointer (`PC`), stack pointer (`SP`), and status (`_Grunning`, `_Grunnable`, `_Gwaiting`, `_Gsyscall`).
- **M (Machine / OS Thread)**: Created via `clone()` / `pthread_create()`. Stack = 8MB.
- **P (Logical Processor)**: Holds local run queue (up to 256 Gs) and memory cache (`mcache`). Number of `P`s is controlled by `runtime.GOMAXPROCS()`.
- **Work Stealing Algorithm**: When a `P` exhausts its local queue, it:
  1. Checks Global Run Queue (GRQ).
  2. Checks Network Poller for ready I/O.
  3. **Steals half the runnable Goroutines** from another randomly chosen `P`'s local run queue!
- **Syscall Preemption & `handoffp`**: When Goroutine $G$ invokes a blocking OS syscall:
  - $M$ enters kernel and blocks.
  - $M$ releases $P$ (`handoffp`), allowing another OS thread ($M$) to bind to $P$ and continue executing remaining Goroutines!

---

## Deep Dive: Memory Virtualization

### Virtual Address Space Layout

```
  0xFFFFFFFF ┌───────────────────────────────────────┐  High Memory (32-bit / 64-bit)
             │  Kernel Space (Protected Ring 0)      │
  0xC0000000 ├───────────────────────────────────────┤
             │  Stack (grows DOWN ↓)                 │  Local variables, return addrs
             │  ↑ RSP (Stack Pointer)                │
             │                  ↓                    │
             │             (Unallocated)             │
             │                  ↑                    │
             │  ↑ Program Break (brk)                │
             │  Heap (grows UP ↑)                    │  Dynamic memory (malloc / make)
  0x00010000 ├───────────────────────────────────────┤
             │  Data & BSS Segment                   │  Globals & static variables
  0x00008000 ├───────────────────────────────────────┤
             │  Code / Text Segment (Read-Only)      │  Executable machine instructions
  0x00000000 └───────────────────────────────────────┘  Low Memory (0x0 Null Pointer Trap)
```

### Stack vs Heap Internals

- **Stack**:
  - Allocation is $O(1)$ by subtracting from Stack Pointer register (`SUB RSP, bytes`).
  - Deallocation is instantaneous upon function return (`ADD RSP, bytes`).
  - Stores activation frames: Function arguments, saved frame pointer (`RBP`), return address, and local variables.
  - **Stack Overflow**: Occurs when deep/infinite recursion exhausts stack boundary.
  - **Go Dynamic Stack Growth**: Go starts goroutines with **2 KB** stacks. When `morestack` detects stack exhaustion, Go allocates a new **2x contiguous block**, copies frames over, fixes internal pointers, and frees the old stack!
- **Heap**:
  - Dynamically managed pool for memory that outlives the creating stack frame (via **Escape Analysis** in Go).
  - Subject to **Fragmentation**.

### Dynamic Memory Allocators

Our simulator implements:
1. **Free-List Allocator (OSTEP Chapter 17)**:
   - Linked list of free blocks.
   - **First-Fit**: Selects first chunk $\ge$ requested size.
   - **Best-Fit**: Scans entire free list to find smallest chunk $\ge$ requested size (minimizes wasted block space).
   - **Coalescing**: When adjacent blocks are freed, merges them into one contiguous free chunk to combat **External Fragmentation**.
2. **Binary Buddy Allocator**:
   - Power-of-two memory block allocation.
   - Recursively divides blocks into left/right "buddies".
   - Upon freeing, immediately merges with buddy if both are free ($O(\log N)$ coalescing).

---

### Hardware Paging & Address Translation

Paging eliminates external fragmentation by dividing virtual memory into fixed-size **Pages** (typically 4 KB) and physical memory into fixed-size **Frames** (PFNs):

$$\text{Virtual Address} = \text{Virtual Page Number (VPN)} + \text{Page Offset}$$

```
Virtual Address:  [ VPN: 20 bits ] [ Offset: 12 bits ]  (for 4KB pages)
                         │
                   Page Table / TLB
                         │
                         ▼
Physical Address: [ PFN: 20 bits ] [ Offset: 12 bits ]
```

#### Page Table Entry (PTE) Hardware Bits:
- **Valid (V)**: Is this page mapped in the process address space?
- **Present (P)**: Is this page in physical RAM (1) or swapped to disk (0)? (Triggers **Page Fault** if 0).
- **Read/Write (R/W)**: Protection permissions (0 = Read-Only, 1 = Read/Write).
- **Accessed / Referenced (A)**: Set by CPU hardware whenever page is read/written (used by Clock replacement).
- **Dirty (D)**: Set by CPU hardware when page is modified (must be written back to swap disk on eviction).
- **PFN**: Physical Frame Number in RAM.

---

### Multi-Level Page Tables

In a 32-bit system with 4KB pages, a linear page table requires $2^{20} = 1,048,576$ entries $\times 4\text{ bytes} = \mathbf{4\text{ MB}}$ per process! In 64-bit systems, linear tables would consume petabytes.

**Hierarchical Two-Level Paging** resolves this:
```
Virtual Address: [ PDE Index: 10 bits ] [ PTE Index: 10 bits ] [ Offset: 12 bits ]
```
- Top-level **Page Directory** has 1024 entries.
- If a 4MB region of memory is unused, the Page Directory Entry (PDE) is marked **Invalid**, and the entire second-level Page Table **is NEVER allocated**!
- Provides **>95% memory savings** for typical sparse address spaces.

---

### Translation Lookaside Buffer (TLB Cache)

Because page table walks require extra RAM accesses for *every single instruction*, CPU MMUs use a specialized hardware cache called the **TLB**:
- **TLB Hit**: Translation found in cache ($1\text{ CPU cycle}$).
- **TLB Miss**: Hardware MMU walks page table in RAM ($100-200\text{ CPU cycles}$), loads entry into TLB.
- **ASID (Address Space Identifier)**: Hardware tag on TLB entries preventing full TLB flushes on process context switches.

---

### Page Faults, Swap Space & Replacement Policies

When a process accesses a page with $\text{Present} = 0$:
1. CPU MMU generates a **Page Fault Interrupt** (Vector 14).
2. OS Kernel traps, selects a physical frame (evicting another page if RAM is full).
3. Reads requested page from **Swap Disk** into RAM.
4. Updates PTE $\text{Present} = 1$, $\text{PFN} = \text{frame}$, restores instruction.

#### Page Replacement Algorithms:
- **FIFO**: Evicts oldest loaded page.
- **LRU (Least Recently Used)**: Evicts page unaccessed for longest time (high hardware overhead).
- **Clock (Second-Chance Algorithm)**:
  - Treats frames as a circular list with a moving hand.
  - If $\text{UseBit} = 1$: Clears bit ($\text{UseBit} = 0$) and advances hand (gives second chance).
  - If $\text{UseBit} = 0$: Evicts this frame!

---

## Deep Dive: Concurrency & Synchronization

### Data Races & Hardware Interleaving

A **Data Race** occurs when two concurrent threads access the same memory location without synchronization, and at least one access is a write.

#### The Assembly-Level Root Cause (Non-Atomic Load-Modify-Store):
```assembly
Step 1 [Thread 1]: mov eax, [counter]   ; Reads counter (e.g. 50) into CPU register
──── CONTEXT SWITCH / HARDWARE INTERLEAVING ────
Step 2 [Thread 2]: mov eax, [counter]   ; Reads same counter (50)
Step 3 [Thread 2]: add eax, 1           ; Increments eax to 51
Step 4 [Thread 2]: mov [counter], eax   ; Writes 51 to RAM
──── CONTEXT SWITCH BACK TO THREAD 1 ────
Step 5 [Thread 1]: add eax, 1           ; Increments its OWN local register (50 + 1 = 51)
Step 6 [Thread 1]: mov [counter], eax   ; OVERWRITES 51 to RAM (LOST THREAD 2's UPDATE!)
```

---

### Hardware Atomics

Hardware atomic instructions lock the CPU cache line (Cache-Coherency Protocol / MESI) to ensure atomic read-modify-write:
- `CompareAndSwap(addr, old, new)` (`LOCK CMPXCHG` on x86)
- `TestAndSet(addr, new)` (`XCHG` on x86)
- `FetchAndAdd(addr, delta)` (`LOCK XADD` on x86)

---

### Spinlocks vs Blocking Mutexes

- **Spinlock**:
  - Loops continuously on atomic CAS until lock is acquired.
  - Fast if critical section is ultra-short ($< 10\text{ns}$).
  - *Problem*: Burns 100% CPU core cycles on long waits and causes priority inversion.
- **Blocking Mutex (Futex)**:
  - **Phase 1 (Fast Path)**: Tries atomic CAS in user-space.
  - **Phase 2 (Slow Path)**: If contended, issues `sys_futex` system call to park thread onto kernel wait queue (`_Gwaiting`). OS deschedules thread until `Unlock()` wakes it up. Zero CPU wasted!

---

### Condition Variables & Mesa Semantics

A **Condition Variable (CV)** allows threads to sleep until a specific shared state condition becomes true.

#### The Mandatory `while` Loop (Mesa Semantics):
```go
mu.Lock()
for !condition {  // MUST BE 'for' / 'while', NEVER 'if'!
    cv.Wait(&mu)
}
// Access protected state safely
mu.Unlock()
```
*Why?*
1. **Spurious Wakeups**: OS signals may wake threads without explicit trigger.
2. **Mesa Semantics**: Between when `Signal()` is sent and the woken thread re-acquires the lock, another thread might have slipped in and invalidated the condition!

---

### Deadlocks, Coffman Conditions & Banker's Algorithm

A **Deadlock** occurs when a set of threads are permanently blocked waiting for resources held by each other.

#### The 4 Coffman Conditions (Must ALL hold for Deadlock):
1. **Mutual Exclusion**: Resources cannot be shared.
2. **Hold and Wait**: Thread holds one resource while requesting another.
3. **No Preemption**: Resources cannot be forcibly taken from a holding thread.
4. **Circular Wait**: Closed chain of threads where $T_1 \to T_2 \to \dots \to T_n \to T_1$.

#### Dijkstra's Banker's Algorithm:
Dynamically evaluates every resource request. A request is granted **only if a Safe Sequence $\langle P_1, P_2, \dots, P_n \rangle$ exists** where every process can complete using remaining Available + Allocated resources.

---

### Dining Philosophers Problem

5 philosophers sit around a table with 5 forks. Each needs 2 forks to eat.
- **Deadlock Case**: Every philosopher picks up left fork simultaneously $\to$ all 5 wait forever for right fork (Circular Wait!).
- **Dijkstra's Resource Ordering Solution**: Order forks by ID ($0 \dots 4$). Always acquire lower-numbered fork first!
  - Philosopher 4 acquires Fork 0 before Fork 4.
  - This mathematically breaks the circular dependency chain, eliminating deadlock completely.

---

## Deep Dive: I/O Models & Event Multiplexing

### Select/Poll $O(N)$ Bottleneck vs Epoll $O(1)$ Architecture

| I/O Model | Syscall | Cost per Event | Kernel Data Structure | Scalability (C10K) |
| :--- | :--- | :--- | :--- | :--- |
| **Blocking I/O** | `read()` | 1 Thread per FD | Kernel Wait Queue | Terrible ($>1000$ threads crashes OS) |
| **Select** | `select()` | $O(N)$ | Bitmask (`fd_set`) | Poor (Max 1024 FDs, scans all $N$) |
| **Poll** | `poll()` | $O(N)$ | Array of `pollfd` | Poor (Linear array copy & scan) |
| **Epoll / Kqueue** | `epoll_wait()` | **$O(1)$** | **Red-Black Tree + Ready-List** | **Superb ($>1,000,000$ concurrent sockets)** |

#### How `epoll` Works Under The Hood:
1. `epoll_ctl()` registers socket in the kernel's **Red-Black Tree** once.
2. When a network packet arrives on the NIC, hardware interrupt calls kernel callback `ep_poll_callback()`.
3. Kernel appends **ONLY the ready socket** to the epoll **Ready List** (Doubly-Linked List).
4. `epoll_wait()` sleeps until interrupt, then returns **only active events in $O(1)$ time** without scanning idle connections.

---

### Go Runtime Netpoller Integration

In Go, you write clean, synchronous code:
```go
conn, _ := listener.Accept()
go func() {
    buf := make([]byte, 1024)
    n, err := conn.Read(buf) // Looks blocking!
}()
```

Under the hood:
1. Go sets socket to non-blocking (`O_NONBLOCK`).
2. When `Read()` returns `EAGAIN`, Go runtime registers FD with the runtime **Netpoller** (`epoll`/`kqueue`).
3. Goroutine $G$ changes to `_Gwaiting` (parks).
4. The OS Thread $M$ **DOES NOT BLOCK**! $M$ immediately executes other Goroutines.
5. When `epoll` signals data is ready, background `sysmon` wakes $G$ and puts it into $P$'s run queue (`_Grunnable`).

---

## Concurrency vs True Parallelism

- **Concurrency**: Dealing with lots of things at once (**Composition & Structure**).
  - A program can be 100% concurrent running on a single 1-core CPU via context switching.
- **Parallelism**: Doing lots of things at once (**Simultaneous Hardware Execution**).
  - Requires multiple physical CPU cores.

#### Amdahl's Law & Lock Contention:
Adding 64 CPU cores **does not guarantee 64x speedup** if goroutines contend heavily on a single mutex! Contention serializes execution, forcing cores to sleep/spin. High-performance systems use lock-free atomics, channels, or partitioned state.

---

## Step-by-Step Interactive Debugger

Launch the interactive step-by-step debugger with:
```bash
./os-sim debug
```

### Available Debugger Commands:
- `step [N]` / `s [N]`: Advance kernel execution by $N$ hardware CPU ticks (triggers timer interrupts and context switches).
- `dash` / `d`: Redraw the live operating system dashboard (CPU, Process Table, TLB hit rates, memory stats).
- `fork` / `f`: Execute `sys_fork()` to clone the current running process.
- `exec <name>` / `e <name>`: Execute `sys_execve()` to load a new executable binary into the running process.
- `kill [PID]` / `exit [PID]`: Terminate a process with exit code 0, turning it into a `ZOMBIE`.
- `wait [parentPID]` / `w`: Call `sys_waitpid()` to reap a zombie child process and destroy its PCB.
- `ps`: Print the active Unix Process Table and parent/child hierarchies.
- `quit` / `q`: Exit the debugger.

---

## Codebase & Package Reference Guide

### `pkg/kernel`
- `HardwareCPU`: Dual-mode CPU execution simulator (Ring 0 vs Ring 3) and register context.
- `TrapTable`: Interrupt Descriptor Table (IDT) routing CPU exceptions, interrupts, and syscall gates.
- `SyscallDispatcher`: System call dispatch engine implementing x86-64 calling conventions and pointer validation.
- `ProcessManager`: Unix process lifecycle engine managing `fork`, `exec`, `exit`, `wait`, zombies, and orphan adoption by `init`.
- `MemoryIsolationLab`: Proves virtual memory isolation and protection faults.
- `OSKernel`: Unified OS kernel coordinating CPU scheduling, memory management, syscalls, and IPC.

### `pkg/ipc`
- `UnixPipe`: Anonymous Unix pipe with in-kernel ring buffer and blocking read/write flow control.
- `SharedMemoryManager`: Zero-copy shared physical frames with IPC semaphores.
- `MessageQueue`: Kernel message queue with typed message passing.

### `pkg/cpu`
- `Process`: Represents Process Control Block (PCB) with states, priority, registers, and metrics.
- `Thread`: Represents Thread Control Block (TCB) with private stack boundaries.
- `ContextSwitcher`: Simulates register save/restore and computes CPU cycle overhead & TLB invalidation cost.
- `FIFOScheduler`, `SJFScheduler`, `STCFScheduler`, `RRScheduler`, `MLFQScheduler`, `LotteryScheduler`: Implementation of all classic CPU scheduling algorithms.
- `GMPRuntime`: Go Runtime M:N scheduler simulator with Work-Stealing, Syscall Handoff, and stack metrics.

### `pkg/memory`
- `AddressSpace`: Virtual address space layout (Code, Data, Heap, Stack, Guard Pages) and Base & Bounds.
- `CallStack`: Stack frame activation records (`RSP`, `RBP`), push/pop mechanics, and stack overflow detection.
- `FreeListAllocator`: Dynamic memory allocator supporting First-Fit, Best-Fit, and block coalescing.
- `BuddyAllocator`: Binary power-of-two buddy allocator.
- `PageTable`: Linear Page Table and MMU address translation.
- `MultiLevelPageTable`: Two-level hierarchical page table (Page Directory + Page Tables).
- `TLB`: Translation Lookaside Buffer hardware cache with hit/miss tracking and LRU eviction.
- `MemoryManager`: Virtual memory manager coordinating page faults, swap disk I/O, and Clock replacement.

### `pkg/concurrency`
- `RunDataRaceDemo`: Live demonstration of data race condition on shared integer.
- `HardwareAtomics`: Hardware-level CAS, XCHG, and Fetch-And-Add primitives.
- `Spinlock`: Busy-waiting spinlock.
- `Mutex`: Yielding/blocking mutex with wait queue (Futex-style).
- `CondVar` & `BoundedBuffer`: Condition Variable implementing Mesa semantics and Producer-Consumer buffer.
- `Semaphore`: Dijkstra's counting semaphore ($P$/Wait and $V$/Signal).
- `RWLock`: Read-Write lock with writer preference.
- `BankersState`: Deadlock avoidance using Dijkstra's Banker's Algorithm.
- `DiningPhilosophers`: Dining Philosophers deadlock reproduction and Resource Hierarchy solution.

### `pkg/io`
- `BlockingIO`: Synchronous blocking I/O simulation.
- `SelectPoll`: $O(N)$ Select/Poll file descriptor scanner.
- `Epoll`: $O(1)$ Epoll event notification simulator with Ready-List and kernel interrupt handling.
- `GoNetpoller`: Go runtime Netpoller and epoll integration.

### `pkg/parallelism`
- `RunComparisonBenchmark`: Hardware benchmark measuring single-core concurrency vs multi-core parallelism and lock contention.

### `pkg/visualizer`
- `SectionHeader`, `SubHeader`, `Box`, `Badge`: Terminal ANSI formatting.
- `RenderGanttChart`: ASCII Gantt chart timeline generator.
- `Table`: Formatted ASCII table generator.
- `RenderAddressSpaceLayout`, `RenderTranslationDiagram`: ASCII memory maps.

---

## License

MIT License. Designed for deep educational exploration of Operating Systems and Go Runtime internals.
