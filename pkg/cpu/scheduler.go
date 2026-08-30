package cpu

import (
	"fmt"
	"os-simulator/pkg/visualizer"
)

// Scheduler defines the common interface implemented by all CPU scheduling algorithms
type Scheduler interface {
	Name() string
	AddProcess(p *Process)
	Tick(currentTime int) (currentRunning *Process, completed bool)
	GetCompleted() []*Process
	GetGantt() []visualizer.GanttSegment
	GetMetrics() Metrics
}

// Metrics computes standard OS scheduling performance benchmarks
type Metrics struct {
	TotalTime           int     `json:"total_time"`
	AvgTurnaroundTime   float64 `json:"avg_turnaround_time"`
	AvgWaitingTime      float64 `json:"avg_waiting_time"`
	AvgResponseTime     float64 `json:"avg_response_time"`
	CPUUtilization      float64 `json:"cpu_utilization"`
	Throughput          float64 `json:"throughput"`
	TotalContextSwitches int     `json:"total_context_switches"`
}

// CalculateMetrics calculates turnaround, waiting, response times from finished processes
func CalculateMetrics(completed []*Process, totalTime, busyTime, contextSwitches int) Metrics {
	n := len(completed)
	if n == 0 {
		return Metrics{}
	}

	var sumTurnaround, sumWaiting, sumResponse int
	for _, p := range completed {
		sumTurnaround += p.Turnaround
		sumWaiting += p.WaitingTime
		sumResponse += p.ResponseTime
	}

	utilization := 0.0
	if totalTime > 0 {
		utilization = (float64(busyTime) / float64(totalTime)) * 100.0
	}

	throughput := 0.0
	if totalTime > 0 {
		throughput = float64(n) / float64(totalTime)
	}

	return Metrics{
		TotalTime:            totalTime,
		AvgTurnaroundTime:    float64(sumTurnaround) / float64(n),
		AvgWaitingTime:       float64(sumWaiting) / float64(n),
		AvgResponseTime:      float64(sumResponse) / float64(n),
		CPUUtilization:       utilization,
		Throughput:           throughput,
		TotalContextSwitches: contextSwitches,
	}
}

// FormatMetricsTable formats scheduling metrics into a visual ASCII table
func FormatMetricsTable(schedName string, m Metrics, completed []*Process) string {
	tbl := visualizer.NewTable("PID", "Process Name", "Arrival", "Burst", "Start", "Finish", "Wait", "Turnaround", "Response")
	tbl.SetAlignment("center", "left", "right", "right", "right", "right", "right", "right", "right")

	for _, p := range completed {
		tbl.AddRow(
			fmt.Sprintf("%d", p.PID),
			p.Name,
			fmt.Sprintf("%d", p.ArrivalTime),
			fmt.Sprintf("%d", p.BurstTime),
			fmt.Sprintf("%d", p.StartTime),
			fmt.Sprintf("%d", p.FinishTime),
			fmt.Sprintf("%d", p.WaitingTime),
			fmt.Sprintf("%d", p.Turnaround),
			fmt.Sprintf("%d", p.ResponseTime),
		)
	}

	summary := []string{
		fmt.Sprintf("Scheduler:            %s%s%s", visualizer.Bold, schedName, visualizer.Reset),
		fmt.Sprintf("Total Duration:        %d ms", m.TotalTime),
		fmt.Sprintf("Avg Turnaround Time:   %.2f ms (OSTEP Metric: Finish - Arrival)", m.AvgTurnaroundTime),
		fmt.Sprintf("Avg Waiting Time:      %.2f ms (Time in READY queue)", m.AvgWaitingTime),
		fmt.Sprintf("Avg Response Time:     %.2f ms (OSTEP Metric: First Run - Arrival)", m.AvgResponseTime),
		fmt.Sprintf("CPU Utilization:       %.1f%%", m.CPUUtilization),
		fmt.Sprintf("Throughput:            %.3f procs/ms", m.Throughput),
		fmt.Sprintf("Total Context Switches:%d", m.TotalContextSwitches),
	}

	return tbl.Render() + "\n" + visualizer.Box("Scheduling Benchmark Summary", summary)
}
