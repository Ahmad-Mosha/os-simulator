package cpu

import (
	"os-simulator/pkg/visualizer"
)

// FIFOScheduler implements First-In, First-Out (FCFS) non-preemptive scheduling
// OSTEP Chapter 7: The simplest scheduling algorithm, vulnerable to the Convoy Effect.
type FIFOScheduler struct {
	allProcesses   []*Process
	readyQueue     []*Process
	running        *Process
	completed      []*Process
	gantt          []visualizer.GanttSegment
	contextSwitches int
	busyTime       int
	totalTime      int
}

func NewFIFOScheduler() *FIFOScheduler {
	return &FIFOScheduler{
		allProcesses: make([]*Process, 0),
		readyQueue:   make([]*Process, 0),
		completed:    make([]*Process, 0),
		gantt:        make([]visualizer.GanttSegment, 0),
	}
}

func (s *FIFOScheduler) Name() string {
	return "FIFO (First-In, First-Out / FCFS)"
}

func (s *FIFOScheduler) AddProcess(p *Process) {
	s.allProcesses = append(s.allProcesses, p)
}

func (s *FIFOScheduler) Tick(currentTime int) (*Process, bool) {
	s.totalTime = currentTime + 1

	// Check newly arriving processes
	for _, p := range s.allProcesses {
		if p.ArrivalTime == currentTime && p.State == StateNew {
			p.State = StateReady
			s.readyQueue = append(s.readyQueue, p)
		}
	}

	// Update waiting time for processes in ready queue
	for _, p := range s.readyQueue {
		p.WaitingTime++
	}

	// If no process is currently running, schedule the next one in FIFO order
	if s.running == nil && len(s.readyQueue) > 0 {
		s.running = s.readyQueue[0]
		s.readyQueue = s.readyQueue[1:]
		s.running.State = StateRunning
		s.contextSwitches++

		if s.running.StartTime == -1 {
			s.running.StartTime = currentTime
			s.running.ResponseTime = currentTime - s.running.ArrivalTime
		}
	}

	if s.running != nil {
		s.busyTime++
		s.running.RemainingTime--

		// Record Gantt chart segment
		s.gantt = append(s.gantt, visualizer.GanttSegment{
			EntityID:  s.running.Name,
			StartTime: currentTime,
			EndTime:   currentTime + 1,
		})

		// Check if running process finished
		if s.running.RemainingTime == 0 {
			s.running.State = StateTerminated
			s.running.FinishTime = currentTime + 1
			s.running.Turnaround = s.running.FinishTime - s.running.ArrivalTime
			s.completed = append(s.completed, s.running)
			s.running = nil
		}
	} else {
		// CPU Idle
		s.gantt = append(s.gantt, visualizer.GanttSegment{
			EntityID:  "IDLE",
			StartTime: currentTime,
			EndTime:   currentTime + 1,
		})
	}

	allDone := len(s.completed) == len(s.allProcesses) && len(s.allProcesses) > 0
	return s.running, allDone
}

func (s *FIFOScheduler) GetCompleted() []*Process {
	return s.completed
}

func (s *FIFOScheduler) GetGantt() []visualizer.GanttSegment {
	return s.gantt
}

func (s *FIFOScheduler) GetMetrics() Metrics {
	return CalculateMetrics(s.completed, s.totalTime, s.busyTime, s.contextSwitches)
}
