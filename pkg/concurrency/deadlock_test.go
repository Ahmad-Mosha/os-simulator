package concurrency

import (
	"testing"
)

func TestBankersAlgorithmSafeState(t *testing.T) {
	// 5 processes, 3 resource types (A:10, B:5, C:7) - Standard Textbook Example
	resNames := []string{"A", "B", "C"}
	total := []int{10, 5, 7}
	b := NewBankersState(5, resNames, total)

	// P0
	b.SetProcessClaim(0, []int{7, 5, 3}, []int{0, 1, 0})
	// P1
	b.SetProcessClaim(1, []int{3, 2, 2}, []int{2, 0, 0})
	// P2
	b.SetProcessClaim(2, []int{9, 0, 2}, []int{3, 0, 2})
	// P3
	b.SetProcessClaim(3, []int{2, 2, 2}, []int{2, 1, 1})
	// P4
	b.SetProcessClaim(4, []int{4, 3, 3}, []int{0, 0, 2})

	isSafe, seq, _ := b.CheckSafeState()
	if !isSafe {
		t.Fatalf("Expected safe state, got unsafe")
	}

	if len(seq) != 5 {
		t.Errorf("Expected 5 processes in safe sequence, got %d", len(seq))
	}

	// Test Request: P1 requests [1, 0, 2]
	granted, _ := b.RequestResources(1, []int{1, 0, 2})
	if !granted {
		t.Errorf("Expected request from P1 to be granted safely")
	}
}

func TestDiningPhilosophersResourceHierarchy(t *testing.T) {
	dp := NewDiningPhilosophers(5)
	dp.RunSafeResourceHierarchy(10)

	for i := 0; i < 5; i++ {
		if dp.MealsEaten[i] != 10 {
			t.Errorf("Expected philosopher %d to eat 10 meals, got %d", i, dp.MealsEaten[i])
		}
	}
}
