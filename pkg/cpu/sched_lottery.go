package cpu

import (
	"fmt"
	"math/rand"
	"os-simulator/pkg/visualizer"
)

// LotteryScheduler implements Proportional-Share Lottery Scheduling
// OSTEP Chapter 9: Probabilistic approach where processes hold tickets representing share of CPU.
type LotteryScheduler struct {
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
	rng            *rand.Rand
}

func NewLotteryScheduler(quantum int, seed int64) *LotteryScheduler {
	if quantum <= 0 {
		quantum = 2
	}
	return &LotteryScheduler{
		Quantum:      quantum,
		allProcesses: make([]*Process, 0),
		readyQueue:   make([]*Process, 0),
		completed:    make([]*Process, 0),
		gantt:        make([]visualizer.GanttSegment, 0),
		rng:          rand.New(rand.NewSource(seed)),
	}
}

func (s *LotteryScheduler) Name() string {
	return fmt.Sprintf("Lottery Scheduler (Proportional Share, Quantum=%d)", s.Quantum)
}

func (s *LotteryScheduler) AddProcess(p *Process) {
	s.allProcesses = append(s.allProcesses, p)
}

func (s *LotteryScheduler) Tick(currentTime int) (*Process, bool) {
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

	if s.running != nil && s.currentSlice >= s.Quantum {
		s.running.State = StateReady
		s.readyQueue = append(s.readyQueue, s.running)
		s.running = nil
		s.currentSlice = 0
	}

	if s.running == nil && len(s.readyQueue) > 0 {
		// Calculate total tickets among ready processes
		totalTickets := 0
		for _, p := range s.readyQueue {
			totalTickets += p.Tickets
		}

		if totalTickets > 0 {
			winningTicket := s.rng.Intn(totalTickets)
			ticketSum := 0
			winnerIdx := 0

			for i, p := range s.readyQueue {
				ticketSum += p.Tickets
				if ticketSum > winningTicket {
					winnerIdx = i
					break
				}
			}

			s.running = s.readyQueue[winnerIdx]
			// Remove winner from ready queue
			s.readyQueue = append(s.readyQueue[:winnerIdx], s.readyQueue[winnerIdx+1:]...)
			s.running.State = StateRunning
			s.currentSlice = 0
			s.contextSwitches++

			if s.running.StartTime == -1 {
				s.running.StartTime = currentTime
				s.running.ResponseTime = currentTime - s.running.ArrivalTime
			}
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

func (s *LotteryScheduler) GetCompleted() []*Process {
	return s.completed
}

func (s *LotteryScheduler) GetGantt() []visualizer.GanttSegment {
	return s.gantt
}

func (s *LotteryScheduler) GetMetrics() Metrics {
	return CalculateMetrics(s.completed, s.totalTime, s.busyTime, s.contextSwitches)
}
