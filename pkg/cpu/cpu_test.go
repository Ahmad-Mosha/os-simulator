package cpu

import (
	"testing"
)

func createTestProcesses() []*Process {
	return []*Process{
		NewProcess(1, "P1", 0, 8),
		NewProcess(2, "P2", 2, 4),
		NewProcess(3, "P3", 4, 2),
	}
}

func runSchedulerToCompletion(s Scheduler, procs []*Process, maxTicks int) Metrics {
	for _, p := range procs {
		s.AddProcess(p.Clone())
	}

	for t := 0; t < maxTicks; t++ {
		_, done := s.Tick(t)
		if done {
			break
		}
	}
	return s.GetMetrics()
}

func TestSchedulersExecution(t *testing.T) {
	schedulers := []Scheduler{
		NewFIFOScheduler(),
		NewSJFScheduler(),
		NewSTCFScheduler(),
		NewRRScheduler(2),
		NewMLFQScheduler(50),
		NewLotteryScheduler(2, 42),
	}

	procs := createTestProcesses()

	for _, s := range schedulers {
		t.Run(s.Name(), func(t *testing.T) {
			m := runSchedulerToCompletion(s, procs, 100)
			completed := s.GetCompleted()

			if len(completed) != len(procs) {
				t.Fatalf("Expected %d completed processes, got %d", len(procs), len(completed))
			}

			if m.AvgTurnaroundTime <= 0 {
				t.Errorf("Expected positive turnaround time, got %f", m.AvgTurnaroundTime)
			}

			if m.AvgResponseTime < 0 {
				t.Errorf("Expected non-negative response time, got %f", m.AvgResponseTime)
			}

			gantt := s.GetGantt()
			if len(gantt) == 0 {
				t.Errorf("Expected non-empty Gantt chart segments")
			}
		})
	}
}

func TestContextSwitcher(t *testing.T) {
	cs := NewContextSwitcher()
	p1 := NewProcess(1, "ProcA", 0, 10)
	p2 := NewProcess(2, "ProcB", 0, 10)

	rec1 := cs.SwitchProcessContext(0, nil, p1)
	if rec1.Type != SwitchProcess || rec1.ToEntity != "P1(ProcA)" {
		t.Errorf("Unexpected switch record: %+v", rec1)
	}

	rec2 := cs.SwitchProcessContext(5, p1, p2)
	if rec2.FromEntity != "P1(ProcA)" || rec2.ToEntity != "P2(ProcB)" {
		t.Errorf("Unexpected switch record: %+v", rec2)
	}

	explanation := cs.ExplainContextSwitch(rec2)
	if len(explanation) == 0 {
		t.Errorf("Expected non-empty explanation")
	}
}
