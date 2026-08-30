package memory

import (
	"errors"
	"fmt"
	"os-simulator/pkg/visualizer"
)

var (
	ErrSegmentationFault = errors.New("segmentation fault: virtual address out of bounds or invalid protection")
	ErrStackOverflow     = errors.New("stack overflow: stack collided with heap or guard page")
)

// SegmentType classifies memory sections
type SegmentType string

const (
	SegmentCode  SegmentType = "CODE"
	SegmentData  SegmentType = "DATA"
	SegmentHeap  SegmentType = "HEAP"
	SegmentStack SegmentType = "STACK"
)

// SegmentDescriptor holds Base & Bounds + permissions for a memory segment
type SegmentDescriptor struct {
	Type        SegmentType
	Base        uint64 // Physical base address
	Limit       uint64 // Size of segment in bytes
	GrowsDown   bool   // True for stack (grows downwards)
	Readable    bool
	Writable    bool
	Executable  bool
}

// AddressSpace represents a process's virtual memory layout
// OSTEP Chapter 13: Address Spaces & Chapter 16: Segmentation
type AddressSpace struct {
	PID          int
	TotalSize    uint64 // e.g. 64KB or 4GB
	PhysicalBase uint64 // Base address in physical RAM

	// Boundaries (Virtual Addresses)
	CodeStart  uint64
	CodeEnd    uint64
	DataStart  uint64
	DataEnd    uint64
	HeapStart  uint64
	HeapBreak  uint64 // Current program break (brk)
	StackTop   uint64 // High address where stack begins
	StackPtr   uint64 // Current stack pointer (grows down)
	GuardPage  uint64

	Segments map[SegmentType]*SegmentDescriptor
}

// NewAddressSpace initializes a standard virtual memory layout for a process
func NewAddressSpace(pid int, totalSize uint64, physicalBase uint64) *AddressSpace {
	if totalSize == 0 {
		totalSize = 64 * 1024 // 64 KB default simulation space
	}

	codeSize := uint64(8 * 1024)  // 8KB code
	dataSize := uint64(4 * 1024)  // 4KB data
	heapInit := uint64(12 * 1024) // Heap starts after data at 12KB
	stackInit := totalSize - 1    // Stack starts at top (0xFFFF)

	as := &AddressSpace{
		PID:          pid,
		TotalSize:    totalSize,
		PhysicalBase: physicalBase,
		CodeStart:    0x0000,
		CodeEnd:      codeSize,
		DataStart:    codeSize,
		DataEnd:      codeSize + dataSize,
		HeapStart:    heapInit,
		HeapBreak:    heapInit,
		StackTop:     stackInit,
		StackPtr:     stackInit,
		GuardPage:    32 * 1024,
		Segments:     make(map[SegmentType]*SegmentDescriptor),
	}

	// Define Hardware Segment Descriptors (OSTEP Chapter 16)
	as.Segments[SegmentCode] = &SegmentDescriptor{
		Type: SegmentCode, Base: physicalBase + 0x0000, Limit: codeSize,
		Readable: true, Writable: false, Executable: true,
	}
	as.Segments[SegmentData] = &SegmentDescriptor{
		Type: SegmentData, Base: physicalBase + codeSize, Limit: dataSize,
		Readable: true, Writable: true, Executable: false,
	}
	as.Segments[SegmentHeap] = &SegmentDescriptor{
		Type: SegmentHeap, Base: physicalBase + heapInit, Limit: 0,
		Readable: true, Writable: true, Executable: false,
	}
	as.Segments[SegmentStack] = &SegmentDescriptor{
		Type: SegmentStack, Base: physicalBase + totalSize, Limit: 0,
		GrowsDown: true, Readable: true, Writable: true, Executable: false,
	}

	return as
}

// TranslateBaseAndBounds translates virtual address to physical address using simple Base & Bounds (OSTEP Chapter 15)
func (as *AddressSpace) TranslateBaseAndBounds(vAddr uint64) (uint64, error) {
	if vAddr >= as.TotalSize {
		return 0, fmt.Errorf("%w: virtual address 0x%X exceeds address space size 0x%X", ErrSegmentationFault, vAddr, as.TotalSize)
	}
	pAddr := as.PhysicalBase + vAddr
	return pAddr, nil
}

// Sbrk simulates the sbrk() system call to expand or contract the heap break
func (as *AddressSpace) Sbrk(increment int64) (uint64, error) {
	oldBreak := as.HeapBreak
	newBreak := uint64(int64(oldBreak) + increment)

	if newBreak < as.HeapStart {
		return 0, fmt.Errorf("invalid sbrk: cannot contract heap below heap start (0x%X)", as.HeapStart)
	}
	if newBreak >= as.StackPtr {
		return 0, fmt.Errorf("%w: heap (0x%X) collided with stack pointer (0x%X)", ErrStackOverflow, newBreak, as.StackPtr)
	}

	as.HeapBreak = newBreak
	as.Segments[SegmentHeap].Limit = newBreak - as.HeapStart
	return oldBreak, nil
}

// RenderMemoryMap renders visual ASCII diagram of address space
func (as *AddressSpace) RenderMemoryMap() string {
	return visualizer.RenderAddressSpaceLayout(as.TotalSize, as.StackTop, as.StackPtr, as.HeapStart, as.HeapBreak, as.DataEnd, as.CodeEnd)
}
