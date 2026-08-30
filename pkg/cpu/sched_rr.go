package cpu

import (
	"fmt"
	"os-simulator/pkg/visualizer"
)

// RRScheduler implements Round Robin preemptive scheduling with a fixed time slice (quantum)
// OSTEP Chapter 7: Solves response time by interleaving execution among all ready processes.
type RRScheduler struct {
	Quantum        int
	allProcesses   []*Process
	readyQueue     []*Process
	running        *Process
	currentSlice   int
	completed      []*Process
	gantt          []visualizer.GanttSegment
	contextSwitches int
	busyTime       int
	totalTime      int
}

func NewRRScheduler(quantum int) *RRScheduler {
	if quantum <= 0 {
		quantum = 2
	}
	return &RRScheduler{
		Quantum:      quantum,
		allProcesses: make([]*Process, 0),
		readyQueue:   make([]*Process, 0),
		completed:    make([]*Process, 0),
		gantt:        make([]visualizer.GanttSegment, 0),
	}
}

func (s *RRScheduler) Name() string {
	return fmt.Sprintf("Round Robin (RR, Quantum = %d)", s.Quantum)
}

func (s *RRScheduler) AddProcess(p *Process) {
	s.allProcesses = append(s.allProcesses, p)
}

func (s *RRScheduler) Tick(currentTime int) (*Process, bool) {
	s.totalTime = currentTime + 1

	// Arriving processes added to ready queue
	for _, p := range s.allProcesses {
		if p.ArrivalTime == currentTime && p.State == StateNew {
			p.State = StateReady
			s.readyQueue = append(s.readyQueue, p)
		}
	}

	for _, p := range s.readyQueue {
		p.WaitingTime++
	}

	// Quantum expired check
	if s.running != nil && s.currentSlice >= s.Quantum {
		// Preempt running process and move to back of ready queue
		s.running.State = StateReady
		s.readyQueue = append(s.readyQueue, s.running)
		s.running = nil
		s.currentSlice = 0
	}

	// If CPU idle, schedule next process from ready queue
	if s.running == nil && len(s.readyQueue) > 0 {
		s.running = s.readyQueue[0]
		s.readyQueue = s.readyQueue[1:]
		s.running.State = StateRunning
		s.currentSlice = 0
		s.contextSwitches++

		if s.running.StartTime == -1 {
			s.running.StartTime = currentTime
			s.running.ResponseTime = currentTime - s.running.ArrivalTime
		}
	}

	if s.running != nil {
		s.busyTime++
		s.currentSlice++
		s.running.RemainingTime--

		s.gantt = append(s.gantt, visualizer.GanttSegment{
			EntityID:  s.running.Name,
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

func (s *RRScheduler) GetCompleted() []*Process {
	return s.completed
}

func (s *RRScheduler) GetGantt() []visualizer.GanttSegment {
	return s.gantt
}

func (s *RRScheduler) GetMetrics() Metrics {
	return CalculateMetrics(s.completed, s.totalTime, s.busyTime, s.contextSwitches)
}
