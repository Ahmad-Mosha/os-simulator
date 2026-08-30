package parallelism

import (
	"fmt"
	"math"
	"os-simulator/pkg/visualizer"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// BenchmarkResult holds timing metrics for concurrency vs parallelism comparisons
type BenchmarkResult struct {
	NumCores            int
	TaskCount           int
	IterationsPerTask   int
	DurationSingleCore  time.Duration
	DurationMultiCore   time.Duration
	SpeedupFactor       float64
	LockContendedDuration time.Duration
	LockFreeDuration    time.Duration
}

// RunComparisonBenchmark runs live CPU-bound calculations under GOMAXPROCS(1) vs GOMAXPROCS(NumCPU)
func RunComparisonBenchmark(taskCount, iters int) BenchmarkResult {
	savedProcs := runtime.GOMAXPROCS(0)
	defer runtime.GOMAXPROCS(savedProcs)

	numCPU := runtime.NumCPU()

	// 1. Single-Core Concurrency (Time-Sliced Interleaving on 1 OS Thread)
	runtime.GOMAXPROCS(1)
	start1 := time.Now()
	var wg1 sync.WaitGroup
	wg1.Add(taskCount)
	for i := 0; i < taskCount; i++ {
		go func() {
			defer wg1.Done()
			cpuIntensiveWork(iters)
		}()
	}
	wg1.Wait()
	durSingle := time.Since(start1)

	// 2. Multi-Core True Parallelism (Simultaneous execution on multiple CPU cores)
	runtime.GOMAXPROCS(numCPU)
	startMulti := time.Now()
	var wgMulti sync.WaitGroup
	wgMulti.Add(taskCount)
	for i := 0; i < taskCount; i++ {
		go func() {
			defer wgMulti.Done()
			cpuIntensiveWork(iters)
		}()
	}
	wgMulti.Wait()
	durMulti := time.Since(startMulti)

	// 3. Lock Contention Benchmark (Amdahl's Law demonstration)
	var mu sync.Mutex
	var contendedCounter int64
	startContended := time.Now()
	var wgContended sync.WaitGroup
	wgContended.Add(taskCount)
	for i := 0; i < taskCount; i++ {
		go func() {
			defer wgContended.Done()
			for j := 0; j < iters/10; j++ {
				mu.Lock()
				contendedCounter++
				mu.Unlock()
			}
		}()
	}
	wgContended.Wait()
	durContended := time.Since(startContended)

	// 4. Lock-Free Atomic Benchmark
	var atomicCounter int64
	startAtomic := time.Now()
	var wgAtomic sync.WaitGroup
	wgAtomic.Add(taskCount)
	for i := 0; i < taskCount; i++ {
		go func() {
			defer wgAtomic.Done()
			for j := 0; j < iters/10; j++ {
				atomic.AddInt64(&atomicCounter, 1)
			}
		}()
	}
	wgAtomic.Wait()
	durAtomic := time.Since(startAtomic)

	speedup := float64(durSingle) / float64(durMulti)

	_ = contendedCounter
	_ = atomicCounter

	return BenchmarkResult{
		NumCores:              numCPU,
		TaskCount:             taskCount,
		IterationsPerTask:     iters,
		DurationSingleCore:    durSingle,
		DurationMultiCore:     durMulti,
		SpeedupFactor:         speedup,
		LockContendedDuration: durContended,
		LockFreeDuration:      durAtomic,
	}
}

func cpuIntensiveWork(iters int) float64 {
	sum := 0.0
	for i := 1; i <= iters; i++ {
		sum += math.Sin(float64(i)) * math.Cos(float64(i))
	}
	return sum
}

// RenderBenchmarkReport formats the live benchmark metrics and theoretical distinction
func RenderBenchmarkReport(res BenchmarkResult) string {
	var sb strings.Builder
	sb.WriteString(visualizer.SubHeader("Concurrency vs Parallelism Live Hardware Benchmark"))

	tbl := visualizer.NewTable("Mode / Strategy", "GOMAXPROCS", "Tasks", "Elapsed Time", "Speedup", "Execution Model")
	tbl.SetAlignment("left", "center", "right", "right", "right", "left")

	tbl.AddRow("Concurrency Only", "1 Core", fmt.Sprintf("%d", res.TaskCount), fmt.Sprintf("%v", res.DurationSingleCore), "1.00x", "Time-sliced interleaving on 1 CPU")
	tbl.AddRow("True Parallelism", fmt.Sprintf("%d Cores", res.NumCores), fmt.Sprintf("%d", res.TaskCount), fmt.Sprintf("%v", res.DurationMultiCore), fmt.Sprintf("%.2fx", res.SpeedupFactor), visualizer.Green("Hardware parallel execution"))
	tbl.AddRow("Mutex Contention", fmt.Sprintf("%d Cores", res.NumCores), fmt.Sprintf("%d", res.TaskCount), fmt.Sprintf("%v", res.LockContendedDuration), "-", visualizer.Red("Serialized by heavy Lock (Amdahl's Law)"))
	tbl.AddRow("Lock-Free Atomics", fmt.Sprintf("%d Cores", res.NumCores), fmt.Sprintf("%d", res.TaskCount), fmt.Sprintf("%v", res.LockFreeDuration), "-", visualizer.Cyan("Hardware cache-line atomic scaling"))

	sb.WriteString(tbl.Render())

	theory := []string{
		"CONCURRENCY vs PARALLELISM (Rob Pike):",
		"• Concurrency is about STRUCTURE: composing independent execution units (goroutines/threads).",
		"  - A program can be 100% concurrent running on a single 1-core CPU via context switching.",
		"• Parallelism is about EXECUTION: running multiple computations at the EXACT same physical instant on multi-core hardware.",
		"",
		"AMDAHL'S LAW & LOCK CONTENTION:",
		"• Adding 64 CPU cores DOES NOT guarantee 64x speedup if tasks contend on shared mutexes!",
		"• Critical sections serialize execution -> multi-core threads spend all their time sleeping/spinning.",
		"• High-performance OS & Go architectures minimize shared state (share memory by communicating via channels).",
	}

	sb.WriteString("\n" + visualizer.Box("Core Architectural Principles", theory))
	return sb.String()
}
