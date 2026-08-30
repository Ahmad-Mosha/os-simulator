package concurrency

import (
	"fmt"
	"os-simulator/pkg/visualizer"
	"strings"
	"sync"
	"time"
)

// DiningPhilosophers simulates the classic Dining Philosophers problem
// OSTEP Chapter 31: Common Concurrency Problems: Dining Philosophers
type DiningPhilosophers struct {
	NumPhilosophers int
	Forks           []*sync.Mutex
	MealsEaten      []int
	Logs            []string
	mu              sync.Mutex
}

func NewDiningPhilosophers(n int) *DiningPhilosophers {
	if n <= 0 {
		n = 5
	}
	forks := make([]*sync.Mutex, n)
	for i := range forks {
		forks[i] = &sync.Mutex{}
	}
	return &DiningPhilosophers{
		NumPhilosophers: n,
		Forks:           forks,
		MealsEaten:      make([]int, n),
		Logs:            make([]string, 0),
	}
}

// RunDeadlockScenario demonstrates circular wait deadlock (everyone picks left fork first)
func (dp *DiningPhilosophers) RunDeadlockScenario(timeoutMs int) bool {
	var wg sync.WaitGroup
	deadlockDetected := make(chan struct{})

	// Barrier to force all philosophers to pick left fork at the exact same moment
	startBarrier := make(chan struct{})

	for i := 0; i < dp.NumPhilosophers; i++ {
		id := i
		leftFork := id
		rightFork := (id + 1) % dp.NumPhilosophers

		wg.Add(1)
		go func() {
			defer wg.Done()
			<-startBarrier

			// Step 1: Pick up Left Fork
			dp.Forks[leftFork].Lock()

			// Delay to ensure all 5 philosophers acquire their left fork simultaneously
			time.Sleep(10 * time.Millisecond)

			// Step 2: Try to acquire Right Fork -> CIRCULAR WAIT DEADLOCK!
			dp.Forks[rightFork].Lock()

			// Eat
			dp.mu.Lock()
			dp.MealsEaten[id]++
			dp.mu.Unlock()

			dp.Forks[rightFork].Unlock()
			dp.Forks[leftFork].Unlock()
		}()
	}

	close(startBarrier)

	go func() {
		wg.Wait()
		close(deadlockDetected)
	}()

	select {
	case <-deadlockDetected:
		return false // Finished without deadlock
	case <-time.After(time.Duration(timeoutMs) * time.Millisecond):
		return true // DEADLOCK CONFIRMED! All goroutines stuck waiting in circular cycle
	}
}

// RunSafeResourceHierarchy solves deadlock by breaking Circular Wait (Dijkstra's Resource Ordering)
// Philosopher 4 picks up Fork 0 before Fork 4 (lower indexed fork first).
func (dp *DiningPhilosophers) RunSafeResourceHierarchy(mealsGoal int) {
	var wg sync.WaitGroup

	for i := 0; i < dp.NumPhilosophers; i++ {
		id := i
		leftFork := id
		rightFork := (id + 1) % dp.NumPhilosophers

		// Resource Ordering: Always acquire lower-numbered mutex first!
		firstFork := leftFork
		secondFork := rightFork
		if firstFork > secondFork {
			firstFork, secondFork = secondFork, firstFork
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			for meal := 0; meal < mealsGoal; meal++ {
				// Think
				time.Sleep(1 * time.Millisecond)

				// Acquire lower index fork first
				dp.Forks[firstFork].Lock()
				dp.Forks[secondFork].Lock()

				// Eat
				dp.mu.Lock()
				dp.MealsEaten[id]++
				dp.mu.Unlock()

				dp.Forks[secondFork].Unlock()
				dp.Forks[firstFork].Unlock()
			}
		}()
	}

	wg.Wait()
}

// RenderDiningStatus visualizes the dining philosophers outcome and theory
func (dp *DiningPhilosophers) RenderDiningStatus() string {
	var sb strings.Builder
	sb.WriteString(visualizer.SubHeader("Dining Philosophers Problem & Solutions (OSTEP Chapter 31)"))

	tbl := visualizer.NewTable("Philosopher", "Left Fork", "Right Fork", "Meals Completed", "Status")
	tbl.SetAlignment("center", "center", "center", "right", "center")

	for i := 0; i < dp.NumPhilosophers; i++ {
		tbl.AddRow(
			fmt.Sprintf("Philosopher %d", i),
			fmt.Sprintf("Fork %d", i),
			fmt.Sprintf("Fork %d", (i+1)%dp.NumPhilosophers),
			fmt.Sprintf("%d", dp.MealsEaten[i]),
			visualizer.Badge("COMPLETED SAFELY", visualizer.BgGreen, visualizer.FgHiWhite),
		)
	}

	sb.WriteString(tbl.Render())

	theory := []string{
		"WHY DIJKSTRA'S RESOURCE HIERARCHY SOLVES DEADLOCK:",
		"• In the Naive approach: Every philosopher picks up Fork(i) then Fork(i+1).",
		"  This creates a directed cycle: P0->F0->P1->F1->P2->F2->P3->F3->P4->F4->P0 (Circular Wait!).",
		"• In the Resource Ordering approach: Mutexes are ordered strictly by integer ID.",
		"  Philosopher 4 acquires Fork 0 FIRST, then Fork 4.",
		"  This breaks the cycle because P4 and P0 now compete for Fork 0.",
		"  One of them wins Fork 0, leaving Fork 4 free for Philosopher 3 to eat and finish!",
		"• Result: Circular Wait condition is IMPOSSIBLE -> Deadlock is mathematically eliminated.",
	}
	sb.WriteString("\n" + visualizer.Box("Mathematical Deadlock Elimination Proof", theory))

	return sb.String()
}
