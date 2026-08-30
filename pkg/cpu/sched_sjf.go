package cpu

import (
	"os-simulator/pkg/visualizer"
	"sort"
)

// SJFScheduler implements Shortest Job First (Non-preemptive)
// OSTEP Chapter 7: Solves convoy effect by running shortest jobs first, but non-preemptive.
type SJFScheduler struct {
	allProcesses   []*Process
	readyQueue     []*Process
	running        *Process
	completed      []*Process
	gantt          []visualizer.GanttSegment
	contextSwitches int
	busyTime       int
	totalTime      int
}

func NewSJFScheduler() *SJFScheduler {
	return &SJFScheduler{
		allProcesses: make([]*Process, 0),
		readyQueue:   make([]*Process, 0),
		completed:    make([]*Process, 0),
		gantt:        make([]visualizer.GanttSegment, 0),
	}
}

func (s *SJFScheduler) Name() string {
	return "SJF (Shortest Job First - Non-Preemptive)"
}

func (s *SJFScheduler) AddProcess(p *Process) {
	s.allProcesses = append(s.allProcesses, p)
}

func (s *SJFScheduler) Tick(currentTime int) (*Process, bool) {
	s.totalTime = currentTime + 1

	for _, p := range s.allProcesses {
		if p.ArrivalTime == currentTime && p.State == StateNew {
			p.State = StateReady
			s.readyQueue = append(s.readyQueue, p)
		}
	}

	for _, p := range s.readyQueue {
		p.WaitingTime++
	}

	// If CPU is free, pick shortest burst time job
	if s.running == nil && len(s.readyQueue) > 0 {
		sort.SliceStable(s.readyQueue, func(i, j int) bool {
			return s.readyQueue[i].BurstTime < s.readyQueue[j].BurstTime
		})
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

func (s *SJFScheduler) GetCompleted() []*Process {
	return s.completed
}

func (s *SJFScheduler) GetGantt() []visualizer.GanttSegment {
	return s.gantt
}

func (s *SJFScheduler) GetMetrics() Metrics {
	return CalculateMetrics(s.completed, s.totalTime, s.busyTime, s.contextSwitches)
}

// STCFScheduler implements Shortest Time-to-Completion First (Preemptive SJF)
// OSTEP Chapter 7: Preempts currently running job if a new job arrives with shorter remaining time.
type STCFScheduler struct {
	allProcesses   []*Process
	readyQueue     []*Process
	running        *Process
	completed      []*Process
	gantt          []visualizer.GanttSegment
	contextSwitches int
	busyTime       int
	totalTime      int
}

func NewSTCFScheduler() *STCFScheduler {
	return &STCFScheduler{
		allProcesses: make([]*Process, 0),
		readyQueue:   make([]*Process, 0),
		completed:    make([]*Process, 0),
		gantt:        make([]visualizer.GanttSegment, 0),
	}
}

func (s *STCFScheduler) Name() string {
	return "STCF (Shortest Time-to-Completion First - Preemptive)"
}

func (s *STCFScheduler) AddProcess(p *Process) {
	s.allProcesses = append(s.allProcesses, p)
}

func (s *STCFScheduler) Tick(currentTime int) (*Process, bool) {
	s.totalTime = currentTime + 1

	// Newly arrived jobs
	for _, p := range s.allProcesses {
		if p.ArrivalTime == currentTime && p.State == StateNew {
			p.State = StateReady
			s.readyQueue = append(s.readyQueue, p)
		}
	}

	for _, p := range s.readyQueue {
		p.WaitingTime++
	}

	// Preemption check: if running process has more remaining time than the shortest ready process
	if len(s.readyQueue) > 0 {
		sort.SliceStable(s.readyQueue, func(i, j int) bool {
			return s.readyQueue[i].RemainingTime < s.readyQueue[j].RemainingTime
		})

		shortestReady := s.readyQueue[0]
		if s.running == nil || shortestReady.RemainingTime < s.running.RemainingTime {
			if s.running != nil {
				s.running.State = StateReady
				s.readyQueue = append(s.readyQueue, s.running)
			}
			s.running = shortestReady
			s.readyQueue = s.readyQueue[1:]
			s.running.State = StateRunning
			s.contextSwitches++

			if s.running.StartTime == -1 {
				s.running.StartTime = currentTime
				s.running.ResponseTime = currentTime - s.running.ArrivalTime
			}
		}
	}

	if s.running != nil {
		s.busyTime++
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

func (s *STCFScheduler) GetCompleted() []*Process {
	return s.completed
}

func (s *STCFScheduler) GetGantt() []visualizer.GanttSegment {
	return s.gantt
}

func (s *STCFScheduler) GetMetrics() Metrics {
	return CalculateMetrics(s.completed, s.totalTime, s.busyTime, s.contextSwitches)
}
