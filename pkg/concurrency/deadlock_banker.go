package concurrency

import (
	"fmt"
	"os-simulator/pkg/visualizer"
	"strings"
)

// BankersState represents system resource state for Dijkstra's Banker's Algorithm
// OSTEP Chapter 32: Common Concurrency Problems: Deadlock Avoidance via Banker's Algorithm
type BankersState struct {
	NumProcesses int
	NumResources int
	ResourceNames []string
	Available     []int       // Vector of length M: available units of each resource
	Max           [][]int     // Matrix N x M: maximum demand of each process
	Allocation    [][]int     // Matrix N x M: currently allocated units
	Need          [][]int     // Matrix N x M: remaining needed units (Max - Allocation)
}

func NewBankersState(procCount int, resNames []string, totalResources []int) *BankersState {
	m := len(resNames)
	avail := make([]int, m)
	copy(avail, totalResources)

	maxMat := make([][]int, procCount)
	allocMat := make([][]int, procCount)
	needMat := make([][]int, procCount)

	for i := 0; i < procCount; i++ {
		maxMat[i] = make([]int, m)
		allocMat[i] = make([]int, m)
		needMat[i] = make([]int, m)
	}

	return &BankersState{
		NumProcesses:  procCount,
		NumResources:  m,
		ResourceNames: resNames,
		Available:     avail,
		Max:           maxMat,
		Allocation:    allocMat,
		Need:          needMat,
	}
}

// SetProcessClaim sets maximum and allocated resources for a process
func (b *BankersState) SetProcessClaim(procIdx int, maxClaim []int, currentAlloc []int) {
	for j := 0; j < b.NumResources; j++ {
		b.Max[procIdx][j] = maxClaim[j]
		b.Allocation[procIdx][j] = currentAlloc[j]
		b.Need[procIdx][j] = maxClaim[j] - currentAlloc[j]
		b.Available[j] -= currentAlloc[j]
	}
}

// CheckSafeState executes Banker's Safety Algorithm to find if system is in a safe state
// Returns isSafe, safeSequence, explanation
func (b *BankersState) CheckSafeState() (bool, []int, string) {
	work := make([]int, b.NumResources)
	copy(work, b.Available)

	finish := make([]bool, b.NumProcesses)
	safeSeq := make([]int, 0, b.NumProcesses)

	var logSteps []string
	logSteps = append(logSteps, fmt.Sprintf("Initial Available Vector: %v", work))

	for len(safeSeq) < b.NumProcesses {
		found := false

		for i := 0; i < b.NumProcesses; i++ {
			if !finish[i] {
				// Check if Need[i] <= Work
				canExecute := true
				for j := 0; j < b.NumResources; j++ {
					if b.Need[i][j] > work[j] {
						canExecute = false
						break
					}
				}

				if canExecute {
					// Process i can finish! Release its allocated resources back to Work
					for j := 0; j < b.NumResources; j++ {
						work[j] += b.Allocation[i][j]
					}
					finish[i] = true
					safeSeq = append(safeSeq, i)
					found = true
					logSteps = append(logSteps, fmt.Sprintf("P%d can satisfy its Need %v <= Available -> Executes & releases resources. New Available: %v",
						i, b.Need[i], work))
					break
				}
			}
		}

		if !found {
			// Unsafe State -> Potential Deadlock!
			logSteps = append(logSteps, visualizer.Red("DEADLOCK RISK: No remaining process can satisfy its resource needs with currently available resources!"))
			return false, nil, strings.Join(logSteps, "\n")
		}
	}

	logSteps = append(logSteps, visualizer.Green(fmt.Sprintf("SAFE STATE CONFIRMED: Safe Execution Sequence: P%v", safeSeq)))
	return true, safeSeq, strings.Join(logSteps, "\n")
}

// RequestResources evaluates whether an immediate resource request can be safely granted
func (b *BankersState) RequestResources(procIdx int, request []int) (bool, string) {
	// 1. Check if request <= Need
	for j := 0; j < b.NumResources; j++ {
		if request[j] > b.Need[procIdx][j] {
			return false, fmt.Sprintf("ERROR: Process P%d requested more than its declared maximum claim (Need: %v)", procIdx, b.Need[procIdx])
		}
	}

	// 2. Check if request <= Available
	for j := 0; j < b.NumResources; j++ {
		if request[j] > b.Available[j] {
			return false, fmt.Sprintf("DENIED: Process P%d must wait. Insufficient resources available (%v requested vs %v available)",
				procIdx, request, b.Available)
		}
	}

	// 3. Pretend to allocate and test safety
	for j := 0; j < b.NumResources; j++ {
		b.Available[j] -= request[j]
		b.Allocation[procIdx][j] += request[j]
		b.Need[procIdx][j] -= request[j]
	}

	isSafe, seq, log := b.CheckSafeState()
	if !isSafe {
		// Rollback hypothetical allocation
		for j := 0; j < b.NumResources; j++ {
			b.Available[j] += request[j]
			b.Allocation[procIdx][j] -= request[j]
			b.Need[procIdx][j] += request[j]
		}
		return false, fmt.Sprintf("REQUEST DENIED: Granting request would put system in an UNSAFE state (Deadlock Hazard)!\n%s", log)
	}

	return true, fmt.Sprintf("REQUEST GRANTED: Safe sequence exists: P%v\n%s", seq, log)
}

// RenderState visualizes the Allocation, Max, Need, and Available matrices
func (b *BankersState) RenderState() string {
	var sb strings.Builder
	sb.WriteString(visualizer.SubHeader("Dijkstra's Banker's Algorithm State Matrix"))

	tbl := visualizer.NewTable("Process", "Allocation", "Max Claim", "Need (Remaining)", "Available")
	tbl.SetAlignment("center", "center", "center", "center", "center")

	availStr := fmt.Sprintf("%v", b.Available)
	for i := 0; i < b.NumProcesses; i++ {
		curAvail := ""
		if i == 0 {
			curAvail = availStr
		}

		tbl.AddRow(
			fmt.Sprintf("P%d", i),
			fmt.Sprintf("%v", b.Allocation[i]),
			fmt.Sprintf("%v", b.Max[i]),
			fmt.Sprintf("%v", b.Need[i]),
			curAvail,
		)
	}

	sb.WriteString(tbl.Render())

	coffman := []string{
		"THE 4 COFFMAN CONDITIONS FOR DEADLOCK (Must ALL hold simultaneously):",
		"1. Mutual Exclusion: Resources cannot be shared simultaneously.",
		"2. Hold and Wait: Process holds one resource while waiting to acquire another.",
		"3. No Preemption: Resources cannot be forcibly taken from a holding process.",
		"4. Circular Wait: A closed loop of processes where each holds a resource needed by the next.",
		"",
		"DEADLOCK PREVENTION vs AVOIDANCE:",
		"• Prevention: Invalidate at least ONE of the 4 Coffman conditions (e.g. Total Resource Ordering).",
		"• Avoidance (Banker's): Dynamically check each request and grant ONLY if a Safe Sequence remains.",
	}
	sb.WriteString("\n" + visualizer.Box("Deadlock Theory & Coffman Conditions", coffman))

	return sb.String()
}
