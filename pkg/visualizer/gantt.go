package visualizer

import (
	"fmt"
	"strings"
)

// GanttSegment represents a continuous block of execution for a process/goroutine
type GanttSegment struct {
	EntityID  string // e.g. "P1", "P2", "G1", "IDLE"
	StartTime int
	EndTime   int
	Color     string // Optional ANSI color code
}

// RenderGanttChart produces an ASCII Gantt timeline chart with time markers
func RenderGanttChart(segments []GanttSegment) string {
	if len(segments) == 0 {
		return "No execution recorded.\n"
	}

	// Group adjacent segments if same entity
	merged := make([]GanttSegment, 0, len(segments))
	for _, seg := range segments {
		if len(merged) > 0 && merged[len(merged)-1].EntityID == seg.EntityID && merged[len(merged)-1].EndTime == seg.StartTime {
			merged[len(merged)-1].EndTime = seg.EndTime
		} else {
			merged = append(merged, seg)
		}
	}

	// Entity color mapping
	palette := []string{FgHiGreen, FgHiYellow, FgHiCyan, FgHiMagenta, FgHiBlue, FgGreen, FgYellow, FgCyan}
	colorMap := make(map[string]string)
	pIdx := 0

	for _, seg := range merged {
		if seg.EntityID == "IDLE" {
			colorMap[seg.EntityID] = FgHiBlack
		} else if _, exists := colorMap[seg.EntityID]; !exists {
			colorMap[seg.EntityID] = palette[pIdx%len(palette)]
			pIdx++
		}
	}

	var topBorder strings.Builder
	var labelRow strings.Builder
	var bottomBorder strings.Builder
	var timeRow strings.Builder

	topBorder.WriteString(FgCyan + "┌")
	labelRow.WriteString(FgCyan + "│" + Reset)
	bottomBorder.WriteString(FgCyan + "└")

	for i, seg := range merged {
		duration := seg.EndTime - seg.StartTime
		// Scale width proportional to duration (minimum 4 chars per block)
		width := duration * 3
		if width < 5 {
			width = 5
		}
		if width > 20 {
			width = 20
		}

		color := colorMap[seg.EntityID]
		if seg.Color != "" {
			color = seg.Color
		}

		// Top
		topBorder.WriteString(strings.Repeat("─", width))
		if i < len(merged)-1 {
			topBorder.WriteString("┬")
		}

		// Label
		lbl := fmt.Sprintf("%s%s%s", color+Bold, seg.EntityID, Reset)
		labelRow.WriteString(padString(lbl, width, "center") + FgCyan + "│" + Reset)

		// Bottom
		bottomBorder.WriteString(strings.Repeat("─", width))
		if i < len(merged)-1 {
			bottomBorder.WriteString("┴")
		}
	}
	topBorder.WriteString("┐" + Reset)
	bottomBorder.WriteString("┘" + Reset)

	// Build timeline markers below
	timeRow.WriteString(FgHiWhite + fmt.Sprintf("%-2d", merged[0].StartTime))
	currentPos := 2

	for _, seg := range merged {
		duration := seg.EndTime - seg.StartTime
		width := duration * 3
		if width < 5 {
			width = 5
		}
		if width > 20 {
			width = 20
		}

		targetPos := currentPos + width + 1
		endStr := fmt.Sprintf("%d", seg.EndTime)
		spacing := targetPos - currentPos - len(endStr)
		if spacing < 1 {
			spacing = 1
		}
		timeRow.WriteString(strings.Repeat(" ", spacing) + endStr)
		currentPos = targetPos
	}
	timeRow.WriteString(Reset)

	return fmt.Sprintf("%s\n%s\n%s\n%s\n",
		topBorder.String(),
		labelRow.String(),
		bottomBorder.String(),
		timeRow.String(),
	)
}
