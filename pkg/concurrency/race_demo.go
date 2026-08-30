package concurrency

import (
	"fmt"
	"os-simulator/pkg/visualizer"
	"strings"
	"sync"
	"sync/atomic"
)

// RaceResult holds results of a live race condition test
type RaceResult struct {
	IterationsPerGoroutine int
	NumGoroutines          int
	ExpectedValue          int64
	UnsafeActualValue      int64
	SafeActualValue        int64
	LostUpdates            int64
}

// RunDataRaceDemo executes parallel goroutines with and without synchronization
func RunDataRaceDemo(numGoroutines, iters int) RaceResult {
	var unsafeCounter int64
	var safeCounter int64

	var wgUnsafe sync.WaitGroup
	wgUnsafe.Add(numGoroutines)

	// 1. Unsafe execution with data race (Load-Modify-Store interleaving)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wgUnsafe.Done()
			for j := 0; j < iters; j++ {
				// Non-atomic read + write -> Data Race!
				temp := unsafeCounter
				// Tiny delay / work to encourage hardware thread interleaving
				temp++
				unsafeCounter = temp
			}
		}()
	}
	wgUnsafe.Wait()

	// 2. Safe execution using Atomic Hardware Primitives (LOCK XADD)
	var wgSafe sync.WaitGroup
	wgSafe.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wgSafe.Done()
			for j := 0; j < iters; j++ {
				atomic.AddInt64(&safeCounter, 1)
			}
		}()
	}
	wgSafe.Wait()

	expected := int64(numGoroutines * iters)
	lost := expected - unsafeCounter

	return RaceResult{
		IterationsPerGoroutine: iters,
		NumGoroutines:          numGoroutines,
		ExpectedValue:          expected,
		UnsafeActualValue:      unsafeCounter,
		SafeActualValue:        safeCounter,
		LostUpdates:            lost,
	}
}

// RenderRaceExplanation visualizes the interleaving that causes lost updates
func RenderRaceExplanation(res RaceResult) string {
	var sb strings.Builder
	sb.WriteString(visualizer.SubHeader("Data Race & Concurrency Interleaving (OSTEP Chapter 26)"))

	tbl := visualizer.NewTable("Execution Mode", "Goroutines", "Iterations/G", "Expected Count", "Actual Count", "Lost Updates", "Status")
	tbl.SetAlignment("left", "right", "right", "right", "right", "right", "center")

	unsafeStatus := visualizer.Badge("DATA RACE DETECTED", visualizer.BgRed, visualizer.FgHiWhite)
	if res.LostUpdates == 0 {
		unsafeStatus = visualizer.Badge("NO RACE OBSERVED", visualizer.BgYellow, visualizer.FgHiWhite)
	}
	safeStatus := visualizer.Badge("SYNCHRONIZED (CORRECT)", visualizer.BgGreen, visualizer.FgHiWhite)

	tbl.AddRow(
		"Unsafe (No Locks/Atomics)",
		fmt.Sprintf("%d", res.NumGoroutines),
		fmt.Sprintf("%d", res.IterationsPerGoroutine),
		fmt.Sprintf("%d", res.ExpectedValue),
		fmt.Sprintf("%s%d%s", visualizer.FgHiRed, res.UnsafeActualValue, visualizer.Reset),
		fmt.Sprintf("%s%d%s", visualizer.FgHiRed, res.LostUpdates, visualizer.Reset),
		unsafeStatus,
	)

	tbl.AddRow(
		"Safe (Hardware Atomics)",
		fmt.Sprintf("%d", res.NumGoroutines),
		fmt.Sprintf("%d", res.IterationsPerGoroutine),
		fmt.Sprintf("%d", res.ExpectedValue),
		fmt.Sprintf("%s%d%s", visualizer.FgHiGreen, res.SafeActualValue, visualizer.Reset),
		"0",
		safeStatus,
	)

	sb.WriteString(tbl.Render())

	assemblyTrace := []string{
		"HARDWARE ROOT CAUSE: Non-Atomic Load-Modify-Store (x86 Assembly):",
		"  Step 1 [Thread 1]: mov eax, [counter]   ; Reads counter (e.g., 50) into CPU register",
		"  ──── CONTEXT SWITCH / HARDWARE INTERLEAVING ────",
		"  Step 2 [Thread 2]: mov eax, [counter]   ; Reads same counter (50)",
		"  Step 3 [Thread 2]: add eax, 1           ; Increments eax to 51",
		"  Step 4 [Thread 2]: mov [counter], eax   ; Writes 51 to RAM",
		"  ──── CONTEXT SWITCH BACK TO THREAD 1 ────",
		"  Step 5 [Thread 1]: add eax, 1           ; Increments its OWN local register (50+1=51)",
		"  Step 6 [Thread 1]: mov [counter], eax   ; OVERWRITES 51 to RAM (Lost Thread 2's update!)",
		"",
		"FIX: Mutual Exclusion (Mutex) or Hardware Atomic Instructions (LOCK CMPXCHG / LOCK XADD).",
	}

	sb.WriteString("\n" + visualizer.Box("Assembly Level Interleaving Breakdown", assemblyTrace))
	return sb.String()
}
