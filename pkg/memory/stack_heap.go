package memory

import (
	"fmt"
	"os-simulator/pkg/visualizer"
	"strings"
)

// StackFrame represents an activation record pushed onto the call stack during a function call
type StackFrame struct {
	FunctionName   string                 `json:"function_name"`
	ReturnAddress  uint64                 `json:"return_address"`
	FramePointer   uint64                 `json:"frame_pointer"`   // BP / RBP
	StackPointer   uint64                 `json:"stack_pointer"`   // SP / RSP
	LocalVariables map[string]interface{} `json:"local_variables"`
	FrameSizeBytes uint64                 `json:"frame_size_bytes"`
}

// CallStack simulates hardware execution call stack
type CallStack struct {
	BaseAddress    uint64
	CurrentSP      uint64
	CurrentBP      uint64
	Frames         []*StackFrame
	MaxDepth       int
	TotalSizeBytes uint64
}

func NewCallStack(baseAddr uint64, totalSize uint64) *CallStack {
	return &CallStack{
		BaseAddress:    baseAddr,
		CurrentSP:      baseAddr,
		CurrentBP:      baseAddr,
		Frames:         make([]*StackFrame, 0),
		MaxDepth:       256,
		TotalSizeBytes: totalSize,
	}
}

// PushFrame pushes a new function activation record (call instruction)
func (cs *CallStack) PushFrame(funcName string, retAddr uint64, frameSize uint64, locals map[string]interface{}) (*StackFrame, error) {
	if len(cs.Frames) >= cs.MaxDepth {
		return nil, fmt.Errorf("%w: max stack recursion depth %d reached", ErrStackOverflow, cs.MaxDepth)
	}

	newSP := cs.CurrentSP - frameSize
	// Check if stack exceeded allocated size (underflow of address space)
	if newSP > cs.CurrentSP || (cs.BaseAddress-newSP) > cs.TotalSizeBytes {
		return nil, fmt.Errorf("%w: stack pointer 0x%X exceeded stack boundary", ErrStackOverflow, newSP)
	}

	frame := &StackFrame{
		FunctionName:   funcName,
		ReturnAddress:  retAddr,
		FramePointer:   cs.CurrentBP,
		StackPointer:   newSP,
		LocalVariables: locals,
		FrameSizeBytes: frameSize,
	}

	cs.CurrentBP = cs.CurrentSP
	cs.CurrentSP = newSP
	cs.Frames = append(cs.Frames, frame)

	return frame, nil
}

// PopFrame pops top activation record on function return (ret instruction)
func (cs *CallStack) PopFrame() (*StackFrame, error) {
	if len(cs.Frames) == 0 {
		return nil, fmt.Errorf("stack underflow: no active frames to pop")
	}

	popped := cs.Frames[len(cs.Frames)-1]
	cs.Frames = cs.Frames[:len(cs.Frames)-1]
	cs.CurrentSP += popped.FrameSizeBytes
	cs.CurrentBP = popped.FramePointer

	return popped, nil
}

// RenderStack renders the stack frames visually
func (cs *CallStack) RenderStack() string {
	var sb strings.Builder
	sb.WriteString(visualizer.SubHeader("Call Stack Frames (High Memory ↓ Low Memory)"))

	if len(cs.Frames) == 0 {
		sb.WriteString("  [Call Stack is currently empty]\n")
		return sb.String()
	}

	tbl := visualizer.NewTable("Frame", "Function", "SP (Top)", "BP (Base)", "Return Addr", "Local Variables")
	tbl.SetAlignment("center", "left", "center", "center", "center", "left")

	for i := len(cs.Frames) - 1; i >= 0; i-- {
		f := cs.Frames[i]
		vars := make([]string, 0)
		for k, v := range f.LocalVariables {
			vars = append(vars, fmt.Sprintf("%s=%v", k, v))
		}
		varStr := strings.Join(vars, ", ")
		if varStr == "" {
			varStr = "-"
		}

		tbl.AddRow(
			fmt.Sprintf("#%d", i),
			f.FunctionName,
			fmt.Sprintf("0x%08X", f.StackPointer),
			fmt.Sprintf("0x%08X", f.FramePointer),
			fmt.Sprintf("0x%08X", f.ReturnAddress),
			varStr,
		)
	}

	sb.WriteString(tbl.Render())

	explanation := []string{
		"STACK vs HEAP MECHANICS (Why Threads Have Private Stacks):",
		"• Stack: Stores local variables, return pointers, and control flow.",
		"  - Allocation is $O(1)$ (just adjust RSP register: SUB RSP, bytes).",
		"  - Deallocation is automatic on return (ADD RSP, bytes). No GC needed.",
		"  - Every thread/goroutine MUST have its own Stack so function calls don't overwrite each other!",
		"• Heap: Stores objects whose lifecycle outlives the creating function frame (Escape Analysis).",
		"  - Shared among all threads in the same process.",
		"  - Requires synchronization (locks) or allocator buckets (tcmalloc / Go mcache).",
		"• Go Stack Growth (Contiguous Stacks):",
		"  - Go starts with tiny 2KB stack per goroutine.",
		"  - On stack overflow check (morestack), Go allocates a 2x contiguous stack,",
		"    copies all frames, fixes internal pointers, and deallocates the old stack!",
	}
	sb.WriteString("\n" + visualizer.Box("Deep Dive: Stack & Heap Internals", explanation))

	return sb.String()
}
