package cpu

import (
	"fmt"
	"math/rand"
	"os-simulator/pkg/visualizer"
	"strings"
	"sync"
)

// GoroutineStatus represents lifecycle state of a Goroutine (G) in the Go runtime
type GoroutineStatus string

const (
	GIdle      GoroutineStatus = "Gidle"
	GRunnable  GoroutineStatus = "Grunnable"
	GRunning   GoroutineStatus = "Grunning"
	GSyscall   GoroutineStatus = "Gsyscall"
	GWaiting   GoroutineStatus = "Gwaiting"
	GDead      GoroutineStatus = "Gdead"
)

// Goroutine (G) represents a user-space lightweight execution thread
type Goroutine struct {
	GID           int             `json:"gid"`
	Name          string          `json:"name"`
	Status        GoroutineStatus `json:"status"`
	StackSize     int             `json:"stack_size"` // Starts at 2KB (2048 bytes), grows dynamically
	PC            uint64          `json:"pc"`
	SP            uint64          `json:"sp"`
	RemainingWork int             `json:"remaining_work"`
	AssignedM     int             `json:"assigned_m"` // -1 if not running
	AssignedP     int             `json:"assigned_p"` // -1 if not running
	IsSyscall     bool            `json:"is_syscall"`
	SyscallTicks  int             `json:"syscall_ticks"`
}

// OSThread (M) represents an operating system kernel thread (created via clone/pthread_create)
type OSThread struct {
	MID       int   `json:"mid"`
	OSStack   int   `json:"os_stack"` // Typically 8MB (8,388,608 bytes)
	CurG      *Goroutine `json:"cur_g"`
	BoundP    *LogicalProcessor `json:"bound_p"`
	IsBlocked bool  `json:"is_blocked"` // Blocked in OS syscall
}

// LogicalProcessor (P) represents the context required to execute Go code (GOMAXPROCS)
type LogicalProcessor struct {
	PID        int          `json:"pid"`
	LocalQueue []*Goroutine `json:"local_queue"` // Up to 256 Goroutines
	RunNext    *Goroutine   `json:"run_next"`    // High priority next slot
	BoundM     *OSThread    `json:"bound_m"`
	TickCount  int          `json:"tick_count"`
}

// GMPRuntime simulates the Go Runtime M:N Scheduler
type GMPRuntime struct {
	GOMAXPROCS      int
	GCount          int
	AllGs           []*Goroutine
	AllMs           []*OSThread
	AllPs           []*LogicalProcessor
	GlobalRunQueue  []*Goroutine
	NetworkPoller   []*Goroutine
	CompletedGs     []*Goroutine
	WorkStealEvents []string
	HandoffEvents   []string
	mu              sync.Mutex
	rng             *rand.Rand
}

// NewGMPRuntime initializes Go M:N runtime simulator
func NewGMPRuntime(gomaxprocs int) *GMPRuntime {
	if gomaxprocs <= 0 {
		gomaxprocs = 2
	}

	rt := &GMPRuntime{
		GOMAXPROCS:      gomaxprocs,
		AllGs:           make([]*Goroutine, 0),
		AllMs:           make([]*OSThread, 0),
		AllPs:           make([]*LogicalProcessor, 0),
		GlobalRunQueue:  make([]*Goroutine, 0),
		NetworkPoller:   make([]*Goroutine, 0),
		CompletedGs:     make([]*Goroutine, 0),
		WorkStealEvents: make([]string, 0),
		HandoffEvents:   make([]string, 0),
		rng:             rand.New(rand.NewSource(99)),
	}

	// Initialize P (Logical Processors) and initial M (OS Threads)
	for i := 0; i < gomaxprocs; i++ {
		p := &LogicalProcessor{
			PID:        i,
			LocalQueue: make([]*Goroutine, 0),
		}
		m := &OSThread{
			MID:     i,
			OSStack: 8 * 1024 * 1024, // 8MB OS stack
			BoundP:  p,
		}
		p.BoundM = m
		rt.AllPs = append(rt.AllPs, p)
		rt.AllMs = append(rt.AllMs, m)
	}

	return rt
}

// SpawnGoroutine creates a new Goroutine and places it in an available P's local queue or GRQ
func (rt *GMPRuntime) SpawnGoroutine(name string, workUnits int, isSyscall bool) *Goroutine {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	rt.GCount++
	g := &Goroutine{
		GID:           rt.GCount,
		Name:          name,
		Status:        GRunnable,
		StackSize:     2048, // 2 KB initial stack
		RemainingWork: workUnits,
		AssignedM:     -1,
		AssignedP:     -1,
		IsSyscall:     isSyscall,
	}
	rt.AllGs = append(rt.AllGs, g)

	// Target P (round robin or random)
	targetP := rt.AllPs[rt.GCount%len(rt.AllPs)]
	if len(targetP.LocalQueue) < 256 {
		targetP.LocalQueue = append(targetP.LocalQueue, g)
	} else {
		// Local queue full -> push to Global Run Queue
		rt.GlobalRunQueue = append(rt.GlobalRunQueue, g)
	}

	return g
}

// Step advances runtime execution by 1 tick
func (rt *GMPRuntime) Step(tick int) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	for _, p := range rt.AllPs {
		p.TickCount++

		// Check if M is currently executing a G
		m := p.BoundM
		if m == nil {
			// Try to acquire an idle M or create new M
			m = rt.acquireM(p)
			p.BoundM = m
			if m != nil {
				m.BoundP = p
			}
		}

		if m != nil && m.CurG != nil {
			g := m.CurG

			// Check if Goroutine entered a blocking syscall
			if g.IsSyscall && g.Status == GRunning && g.SyscallTicks == 0 {
				g.Status = GSyscall
				m.IsBlocked = true
				// HANDOFF P: OS thread M blocks in kernel, but P is released so other Gs can run!
				event := fmt.Sprintf("[Tick %d] Handoff: G%d made syscall -> M%d blocked, P%d handed off to new M", tick, g.GID, m.MID, p.PID)
				rt.HandoffEvents = append(rt.HandoffEvents, event)

				// Detach P from M
				p.BoundM = nil
				m.BoundP = nil

				// Find or create a new M for P
				newM := &OSThread{
					MID:     len(rt.AllMs),
					OSStack: 8 * 1024 * 1024,
					BoundP:  p,
				}
				rt.AllMs = append(rt.AllMs, newM)
				p.BoundM = newM
				continue
			}

			// If G is executing normal work
			if g.Status == GRunning {
				g.RemainingWork--
				if g.RemainingWork <= 0 {
					g.Status = GDead
					rt.CompletedGs = append(rt.CompletedGs, g)
					m.CurG = nil
				}
			}
		}

		// If M has no G to run, find work using Go's Work Search Algorithm:
		// 1. Check Global Run Queue every 61 ticks (anti-starvation)
		// 2. Check P's Local Run Queue (runnext then local queue)
		// 3. Check Global Run Queue
		// 4. Work Stealing from other P's local queue (steal half)
		if p.BoundM != nil && p.BoundM.CurG == nil {
			nextG := rt.findRunnableG(p, tick)
			if nextG != nil {
				nextG.Status = GRunning
				nextG.AssignedM = p.BoundM.MID
				nextG.AssignedP = p.PID
				p.BoundM.CurG = nextG
			}
		}
	}

	// Update any blocked syscall Gs
	for _, m := range rt.AllMs {
		if m.IsBlocked && m.CurG != nil {
			m.CurG.SyscallTicks++
			if m.CurG.SyscallTicks >= 3 { // Syscall completes
				m.IsBlocked = false
				m.CurG.Status = GRunnable
				m.CurG.IsSyscall = false
				// Place awakened G into Global Queue
				rt.GlobalRunQueue = append(rt.GlobalRunQueue, m.CurG)
				m.CurG = nil
			}
		}
	}
}

func (rt *GMPRuntime) findRunnableG(p *LogicalProcessor, tick int) *Goroutine {
	// 1. Check GRQ every 61 ticks
	if p.TickCount%61 == 0 && len(rt.GlobalRunQueue) > 0 {
		g := rt.GlobalRunQueue[0]
		rt.GlobalRunQueue = rt.GlobalRunQueue[1:]
		return g
	}

	// 2. Check RunNext slot
	if p.RunNext != nil {
		g := p.RunNext
		p.RunNext = nil
		return g
	}

	// 3. Check Local Run Queue
	if len(p.LocalQueue) > 0 {
		g := p.LocalQueue[0]
		p.LocalQueue = p.LocalQueue[1:]
		return g
	}

	// 4. Check Global Run Queue if local was empty
	if len(rt.GlobalRunQueue) > 0 {
		g := rt.GlobalRunQueue[0]
		rt.GlobalRunQueue = rt.GlobalRunQueue[1:]
		return g
	}

	// 5. Work Stealing: Steal half the work from another randomly chosen P
	numPs := len(rt.AllPs)
	if numPs > 1 {
		startIdx := rt.rng.Intn(numPs)
		for i := 0; i < numPs; i++ {
			victimIdx := (startIdx + i) % numPs
			victimP := rt.AllPs[victimIdx]
			if victimP.PID != p.PID && len(victimP.LocalQueue) > 0 {
				stealCount := (len(victimP.LocalQueue) + 1) / 2
				stolen := victimP.LocalQueue[len(victimP.LocalQueue)-stealCount:]
				victimP.LocalQueue = victimP.LocalQueue[:len(victimP.LocalQueue)-stealCount]

				event := fmt.Sprintf("[Tick %d] Work Stealing: P%d stole %d Goroutine(s) from P%d", tick, p.PID, len(stolen), victimP.PID)
				rt.WorkStealEvents = append(rt.WorkStealEvents, event)

				g := stolen[0]
				p.LocalQueue = append(p.LocalQueue, stolen[1:]...)
				return g
			}
		}
	}

	return nil
}

func (rt *GMPRuntime) acquireM(p *LogicalProcessor) *OSThread {
	for _, m := range rt.AllMs {
		if !m.IsBlocked && m.BoundP == nil {
			return m
		}
	}
	return nil
}

// RenderStatus generates a formatted dashboard of the GMP runtime state
func (rt *GMPRuntime) RenderStatus() string {
	var sb strings.Builder

	sb.WriteString(visualizer.SubHeader(fmt.Sprintf("Go GMP M:N Runtime Model (GOMAXPROCS=%d)", rt.GOMAXPROCS)))

	// Processors & Threads Table
	tbl := visualizer.NewTable("Logical Processor (P)", "Bound OS Thread (M)", "Current Goroutine (G)", "Local Queue (LRQ)")
	tbl.SetAlignment("center", "center", "center", "left")

	for _, p := range rt.AllPs {
		mStr := "None (Idle)"
		gStr := "None (Idle)"
		if p.BoundM != nil {
			mStr = fmt.Sprintf("M%d [OS Stack: 8MB]", p.BoundM.MID)
			if p.BoundM.CurG != nil {
				gStr = fmt.Sprintf("G%d (%s) [Stack: 2KB]", p.BoundM.CurG.GID, p.BoundM.CurG.Name)
			}
		}

		lrqGs := make([]string, 0)
		for _, g := range p.LocalQueue {
			lrqGs = append(lrqGs, fmt.Sprintf("G%d", g.GID))
		}
		lrqStr := strings.Join(lrqGs, ", ")
		if lrqStr == "" {
			lrqStr = "(empty)"
		}

		tbl.AddRow(fmt.Sprintf("P%d", p.PID), mStr, gStr, lrqStr)
	}

	sb.WriteString(tbl.Render())

	// Memory & Stack comparison box
	totalGMem := len(rt.AllGs) * 2 // in KB
	totalMMem := len(rt.AllMs) * 8 // in MB

	comparisonLines := []string{
		fmt.Sprintf("Active Goroutines (G): %d  ──► Total Stack Memory: ~%d KB (2KB initial per G)", len(rt.AllGs), totalGMem),
		fmt.Sprintf("Active OS Threads (M): %d  ──► Total Stack Memory: ~%d MB (8MB fixed per M)", len(rt.AllMs), totalMMem),
		fmt.Sprintf("Global Run Queue (GRQ): %d Goroutine(s) waiting", len(rt.GlobalRunQueue)),
		fmt.Sprintf("Work Stealing Events:   %d", len(rt.WorkStealEvents)),
		fmt.Sprintf("Syscall Handoff Events: %d", len(rt.HandoffEvents)),
		"",
		"WHY GOROUTINES ARE SO FAST & LIGHTWEIGHT:",
		"• 1 Million Goroutines = ~2 GB RAM (Dynamically grows/shrinks via stack copying)",
		"• 1 Million OS Threads = ~8,000 GB RAM (8 TB) -> OS would immediately run Out of Memory (OOM)!",
		"• Goroutine switch cost: ~10-20 nanoseconds (user-space, 3 registers: PC, SP, DX)",
		"• OS Thread switch cost: ~1-2 microseconds (kernel trap, TLB invalidation, CPU pipelines flushed)",
	}

	sb.WriteString("\n" + visualizer.Box("GMP Runtime Memory & Architectural Comparison", comparisonLines))

	if len(rt.WorkStealEvents) > 0 {
		sb.WriteString("\n" + visualizer.Box("Recent Work-Stealing Logs", rt.WorkStealEvents[max(0, len(rt.WorkStealEvents)-4):]))
	}

	return sb.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
