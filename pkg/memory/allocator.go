package memory

import (
	"errors"
	"fmt"
	"os-simulator/pkg/visualizer"
	"strings"
)

var (
	ErrOutOfMemory = errors.New("out of memory: insufficient contiguous memory")
	ErrInvalidFree = errors.New("invalid free: address was not allocated")
)

// BlockHeader represents header metadata for a dynamic heap block
type BlockHeader struct {
	Addr       uint64
	Size       uint64
	IsAllocated bool
	Next       *BlockHeader
	Prev       *BlockHeader
}

// FreeListAllocator implements dynamic memory allocation with Free-List (OSTEP Chapter 17)
type FreeListAllocator struct {
	HeapStart       uint64
	TotalSize       uint64
	Head            *BlockHeader
	AllocatedBytes  uint64
	AllocationsCount int
}

func NewFreeListAllocator(startAddr, totalSize uint64) *FreeListAllocator {
	head := &BlockHeader{
		Addr:        startAddr,
		Size:        totalSize,
		IsAllocated: false,
	}
	return &FreeListAllocator{
		HeapStart: startAddr,
		TotalSize: totalSize,
		Head:      head,
	}
}

// Malloc allocates memory using First-Fit or Best-Fit
func (fla *FreeListAllocator) Malloc(size uint64, bestFit bool) (uint64, error) {
	if size == 0 {
		return 0, nil
	}

	var chosen *BlockHeader
	curr := fla.Head

	if bestFit {
		// Best-Fit: Search entire free list to find smallest block >= requested size
		for curr != nil {
			if !curr.IsAllocated && curr.Size >= size {
				if chosen == nil || curr.Size < chosen.Size {
					chosen = curr
				}
			}
			curr = curr.Next
		}
	} else {
		// First-Fit: Pick first block >= requested size
		for curr != nil {
			if !curr.IsAllocated && curr.Size >= size {
				chosen = curr
				break
			}
			curr = curr.Next
		}
	}

	if chosen == nil {
		return 0, fmt.Errorf("%w: cannot allocate %d bytes (external fragmentation or heap exhaustion)", ErrOutOfMemory, size)
	}

	// Split block if remaining space is useful (e.g. > 16 bytes)
	if chosen.Size > size+16 {
		remaining := &BlockHeader{
			Addr:        chosen.Addr + size,
			Size:        chosen.Size - size,
			IsAllocated: false,
			Next:        chosen.Next,
			Prev:        chosen,
		}
		if chosen.Next != nil {
			chosen.Next.Prev = remaining
		}
		chosen.Next = remaining
		chosen.Size = size
	}

	chosen.IsAllocated = true
	fla.AllocatedBytes += chosen.Size
	fla.AllocationsCount++
	return chosen.Addr, nil
}

// Free deallocates block and coalesces adjacent free blocks (OSTEP Chapter 17)
func (fla *FreeListAllocator) Free(addr uint64) error {
	curr := fla.Head
	for curr != nil {
		if curr.Addr == addr {
			if !curr.IsAllocated {
				return fmt.Errorf("%w: double free at 0x%X", ErrInvalidFree, addr)
			}
			curr.IsAllocated = false
			fla.AllocatedBytes -= curr.Size

			// Coalesce with Next if free
			if curr.Next != nil && !curr.Next.IsAllocated {
				curr.Size += curr.Next.Size
				curr.Next = curr.Next.Next
				if curr.Next != nil {
					curr.Next.Prev = curr
				}
			}

			// Coalesce with Prev if free
			if curr.Prev != nil && !curr.Prev.IsAllocated {
				curr.Prev.Size += curr.Size
				curr.Prev.Next = curr.Next
				if curr.Next != nil {
					curr.Next.Prev = curr.Prev
				}
			}
			return nil
		}
		curr = curr.Next
	}
	return fmt.Errorf("%w: pointer 0x%X not found", ErrInvalidFree, addr)
}

// RenderHeapMap visualizes the free-list memory blocks and fragmentation
func (fla *FreeListAllocator) RenderHeapMap() string {
	var sb strings.Builder
	sb.WriteString(visualizer.SubHeader("Heap Allocator Free-List Blocks"))

	tbl := visualizer.NewTable("Address Range", "Size (Bytes)", "Status", "Block Type")
	tbl.SetAlignment("center", "right", "center", "left")

	var freeBytes, freeBlocks, allocBlocks uint64
	curr := fla.Head

	for curr != nil {
		status := visualizer.Badge("ALLOCATED", visualizer.BgYellow, visualizer.FgHiWhite)
		bType := "In Use (Object payload)"
		if !curr.IsAllocated {
			status = visualizer.Badge("FREE", visualizer.BgGreen, visualizer.FgHiWhite)
			bType = "Free Chunk (Available)"
			freeBytes += curr.Size
			freeBlocks++
		} else {
			allocBlocks++
		}

		tbl.AddRow(
			fmt.Sprintf("0x%08X - 0x%08X", curr.Addr, curr.Addr+curr.Size-1),
			fmt.Sprintf("%d", curr.Size),
			status,
			bType,
		)
		curr = curr.Next
	}

	sb.WriteString(tbl.Render())

	fragLines := []string{
		fmt.Sprintf("Total Heap Size:       %d bytes", fla.TotalSize),
		fmt.Sprintf("Currently Allocated:   %d bytes (%d blocks)", fla.AllocatedBytes, allocBlocks),
		fmt.Sprintf("Currently Free:        %d bytes (%d free chunks)", freeBytes, freeBlocks),
		fmt.Sprintf("External Fragmentation: %d fragmented free chunks", freeBlocks),
		"",
		"FRAGMENTATION INSIGHTS (OSTEP Chapter 17):",
		"• External Fragmentation: Free space is chopped into tiny disjointed pieces.",
		"  Even if total free memory > requested size, allocation fails if no single chunk is contiguous!",
		"• Coalescing: When neighboring blocks are freed, the allocator merges them back into one large chunk.",
	}
	sb.WriteString("\n" + visualizer.Box("Heap Memory Stats & Fragmentation Analysis", fragLines))

	return sb.String()
}

// BuddyNode represents a power-of-two memory block in the Buddy Allocator
type BuddyNode struct {
	Addr       uint64
	Order      int  // Block size = 2^Order
	IsAllocated bool
	Left       *BuddyNode
	Right      *BuddyNode
}

// BuddyAllocator implements Binary Buddy Allocation (OSTEP Chapter 17)
type BuddyAllocator struct {
	BaseAddr uint64
	MinOrder int
	MaxOrder int
	Root     *BuddyNode
}

func NewBuddyAllocator(baseAddr uint64, minOrder, maxOrder int) *BuddyAllocator {
	return &BuddyAllocator{
		BaseAddr: baseAddr,
		MinOrder: minOrder,
		MaxOrder: maxOrder,
		Root: &BuddyNode{
			Addr:  baseAddr,
			Order: maxOrder,
		},
	}
}

// Allocate finds or splits a block of order >= requested order
func (ba *BuddyAllocator) Allocate(size uint64) (uint64, error) {
	targetOrder := ba.MinOrder
	for (1 << targetOrder) < size {
		targetOrder++
	}
	if targetOrder > ba.MaxOrder {
		return 0, fmt.Errorf("%w: requested size %d exceeds max buddy capacity %d", ErrOutOfMemory, size, 1<<ba.MaxOrder)
	}

	node := ba.findAndSplit(ba.Root, targetOrder)
	if node == nil {
		return 0, fmt.Errorf("%w: no buddy block available for order %d (%d bytes)", ErrOutOfMemory, targetOrder, 1<<targetOrder)
	}

	node.IsAllocated = true
	return node.Addr, nil
}

func (ba *BuddyAllocator) findAndSplit(node *BuddyNode, targetOrder int) *BuddyNode {
	if node.IsAllocated {
		return nil
	}

	if node.Left == nil && node.Right == nil {
		if node.Order == targetOrder {
			return node
		}
		if node.Order > targetOrder {
			// Split node into two buddies of Order - 1
			childOrder := node.Order - 1
			childSize := uint64(1 << childOrder)
			node.Left = &BuddyNode{Addr: node.Addr, Order: childOrder}
			node.Right = &BuddyNode{Addr: node.Addr + childSize, Order: childOrder}
			return ba.findAndSplit(node.Left, targetOrder)
		}
		return nil
	}

	if res := ba.findAndSplit(node.Left, targetOrder); res != nil {
		return res
	}
	return ba.findAndSplit(node.Right, targetOrder)
}
