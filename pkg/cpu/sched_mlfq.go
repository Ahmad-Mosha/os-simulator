package cpu

import (
	"fmt"
	"os-simulator/pkg/visualizer"
)

// MLFQQueueConfig defines settings for a single priority level in MLFQ
type MLFQQueueConfig struct {
	Level     int // 0 is highest priority
	Quantum   int // Time slice in ticks
	Allotment int // Total time before demotion to next level
}

// MLFQScheduler implements Multi-Level Feedback Queue with the 5 OSTEP Rules
// OSTEP Chapter 8: Multi-Level Feedback Queue
// - Rule 1: If Priority(A) > Priority(B), A runs (B does not).
// - Rule 2: If Priority(A) == Priority(B), A & B run in Round-Robin using queue's quantum.
// - Rule 3: When a job enters the system, it is placed at the highest priority queue (Q0).
// - Rule 4: Once a job uses up its time allotment at a given level, its priority is reduced (moves down).
// - Rule 5: After some time period S (BoostPeriod), move ALL jobs to topmost queue (Priority Boost).
type MLFQScheduler struct {
	NumQueues       int
	Configs         []MLFQQueueConfig
	Queues          [][]*Process
	BoostPeriod     int // S ticks
	ticksSinceBoost int
	
	allProcesses    []*Process
	running         *Process
	currentSlice    int // Ticks in current RR quantum
	completed       []*Process
	gantt           []visualizer.GanttSegment
	contextSwitches int
	busyTime        int
	totalTime       int
	BoostEvents     []int // Timestamps when priority boost occurred
}

// NewMLFQScheduler creates an MLFQ scheduler with default 3-tier configuration
func NewMLFQScheduler(boostPeriod int) *MLFQScheduler {
	if boostPeriod <= 0 {
		boostPeriod = 50 // Default priority boost every 50 ticks
	}

	configs := []MLFQQueueConfig{
		{Level: 0, Quantum: 2, Allotment: 4},   // Q0: Short quantum, quick response for interactive jobs
		{Level: 1, Quantum: 4, Allotment: 8},   // Q1: Medium quantum
		{Level: 2, Quantum: 8, Allotment: 16},  // Q2: Long quantum / batch processing
	}

	queues := make([][]*Process, len(configs))
	for i := range queues {
		queues[i] = make([]*Process, 0)
	}

	return &MLFQScheduler{
		NumQueues:    len(configs),
		Configs:      configs,
		Queues:       queues,
		BoostPeriod:  boostPeriod,
		allProcesses: make([]*Process, 0),
		completed:    make([]*Process, 0),
		gantt:        make([]visualizer.GanttSegment, 0),
		BoostEvents:  make([]int, 0),
	}
}

func (s *MLFQScheduler) Name() string {
	return fmt.Sprintf("MLFQ (Multi-Level Feedback Queue, %d Queues, Boost=%d)", s.NumQueues, s.BoostPeriod)
}

func (s *MLFQScheduler) AddProcess(p *Process) {
	s.allProcesses = append(s.allProcesses, p)
}

func (s *MLFQScheduler) Tick(currentTime int) (*Process, bool) {
	s.totalTime = currentTime + 1
	s.ticksSinceBoost++

	// Rule 5: Priority Boost (move all active/blocked jobs to Q0 and reset allotment)
	if s.ticksSinceBoost >= s.BoostPeriod {
		s.BoostEvents = append(s.BoostEvents, currentTime)
		s.ticksSinceBoost = 0

		// Collect all processes from all queues
		allReady := make([]*Process, 0)
		for q := 0; q < s.NumQueues; q++ {
			allReady = append(allReady, s.Queues[q]...)
			s.Queues[q] = make([]*Process, 0)
		}

		// Reset all to Q0
		for _, p := range allReady {
			p.CurrentQueue = 0
			p.AllotmentUsed = 0
			s.Queues[0] = append(s.Queues[0], p)
		}

		if s.running != nil {
			s.running.CurrentQueue = 0
			s.running.AllotmentUsed = 0
		}
	}

	// Rule 3: Newly arriving processes enter topmost queue (Q0)
	for _, p := range s.allProcesses {
		if p.ArrivalTime == currentTime && p.State == StateNew {
			p.State = StateReady
			p.CurrentQueue = 0
			p.AllotmentUsed = 0
			s.Queues[0] = append(s.Queues[0], p)
		}
	}

	// Update waiting time for all queued processes
	for q := 0; q < s.NumQueues; q++ {
		for _, p := range s.Queues[q] {
			p.WaitingTime++
		}
	}

	// Check if running process exceeded quantum or allotment
	if s.running != nil {
		qCfg := s.Configs[s.running.CurrentQueue]

		// Rule 4: Allotment check (Accounting against gaming the scheduler)
		if s.running.AllotmentUsed >= qCfg.Allotment {
			// Demote to next queue down (if not already at lowest)
			if s.running.CurrentQueue < s.NumQueues-1 {
				s.running.CurrentQueue++
			}
			s.running.AllotmentUsed = 0
			s.running.State = StateReady
			s.Queues[s.running.CurrentQueue] = append(s.Queues[s.running.CurrentQueue], s.running)
			s.running = nil
			s.currentSlice = 0
		} else if s.currentSlice >= qCfg.Quantum {
			// Quantum expired: stay in same queue level, move to back of queue (Rule 2)
			s.running.State = StateReady
			s.Queues[s.running.CurrentQueue] = append(s.Queues[s.running.CurrentQueue], s.running)
			s.running = nil
			s.currentSlice = 0
		}
	}

	// Rule 1: Schedule highest priority non-empty queue
	if s.running == nil {
		for q := 0; q < s.NumQueues; q++ {
			if len(s.Queues[q]) > 0 {
				s.running = s.Queues[q][0]
				s.Queues[q] = s.Queues[q][1:]
				s.running.State = StateRunning
				s.currentSlice = 0
				s.contextSwitches++

				if s.running.StartTime == -1 {
					s.running.StartTime = currentTime
					s.running.ResponseTime = currentTime - s.running.ArrivalTime
				}
				break
			}
		}
	}

	if s.running != nil {
		s.busyTime++
		s.currentSlice++
		s.running.AllotmentUsed++
		s.running.RemainingTime--

		s.gantt = append(s.gantt, visualizer.GanttSegment{
			EntityID:  fmt.Sprintf("%s(Q%d)", s.running.Name, s.running.CurrentQueue),
			StartTime: currentTime,
			EndTime:   currentTime + 1,
		})

		if s.running.RemainingTime == 0 {
			s.running.State = StateTerminated
			s.running.FinishTime = currentTime + 1
			s.running.Turnaround = s.running.FinishTime - s.running.ArrivalTime
			s.completed = append(s.completed, s.running)
			s.running = nil
			s.currentSlice = 0
		}
	} else {
		s.gantt = append(s.gantt, visualizer.GanttSegment{
			EntityID:  "IDLE",
			StartTime: currentTime,
			EndTime:   currentTime + 1,
		})
	}

	allDone := len(s.completed) == len(s.allProcesses) && len(s.allProcesses) > 0
	return s.running, allDone
}

func (s *MLFQScheduler) GetCompleted() []*Process {
	return s.completed
}

func (s *MLFQScheduler) GetGantt() []visualizer.GanttSegment {
	return s.gantt
}

func (s *MLFQScheduler) GetMetrics() Metrics {
	return CalculateMetrics(s.completed, s.totalTime, s.busyTime, s.contextSwitches)
}
