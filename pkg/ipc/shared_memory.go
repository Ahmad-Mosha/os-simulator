package ipc

import (
	"fmt"
	"os-simulator/pkg/concurrency"
	"os-simulator/pkg/memory"
	"os-simulator/pkg/visualizer"
	"strings"
	"sync"
)

// SharedMemorySegment represents a region of physical RAM mapped into multiple processes
type SharedMemorySegment struct {
	ShmID        int
	PhysicalPFN  uint64
	SizeBytes    int
	PhysicalData []byte
	Semaphore    *concurrency.Semaphore // IPC Semaphore for synchronized access
	mu           sync.Mutex
}

// ProcessMapping records where a shared segment is attached in a process's virtual address space
type ProcessMapping struct {
	PID            int
	VirtualAddress uint64
	PageTable      *memory.PageTable
}

// SharedMemoryManager manages POSIX/System V shared memory segments
type SharedMemoryManager struct {
	Segments map[int]*SharedMemorySegment
	Mappings map[int][]ProcessMapping // ShmID -> list of process mappings
	NextShmID int
	mu       sync.Mutex
}

func NewSharedMemoryManager() *SharedMemoryManager {
	return &SharedMemoryManager{
		Segments:  make(map[int]*SharedMemorySegment),
		Mappings:  make(map[int][]ProcessMapping),
		NextShmID: 1,
	}
}

// CreateSegment allocates a shared physical memory segment (shmget)
func (smm *SharedMemoryManager) CreateSegment(sizeBytes int) (*SharedMemorySegment, error) {
	smm.mu.Lock()
	defer smm.mu.Unlock()

	shmID := smm.NextShmID
	smm.NextShmID++

	seg := &SharedMemorySegment{
		ShmID:        shmID,
		PhysicalPFN:  uint64(500 + shmID), // Dedicated physical frame
		SizeBytes:    sizeBytes,
		PhysicalData: make([]byte, sizeBytes),
		Semaphore:    concurrency.NewSemaphore(1), // Binary semaphore for mutual exclusion
	}

	smm.Segments[shmID] = seg
	smm.Mappings[shmID] = make([]ProcessMapping, 0)
	return seg, nil
}

// Attach maps the shared physical frame into a process's virtual address space (shmat)
func (smm *SharedMemoryManager) Attach(shmID int, pid int, vAddr uint64, pt *memory.PageTable) error {
	smm.mu.Lock()
	defer smm.mu.Unlock()

	seg, exists := smm.Segments[shmID]
	if !exists {
		return fmt.Errorf("shmat: invalid shmid %d", shmID)
	}

	// Map VPN -> PhysicalPFN in process's page table
	vpn := vAddr / pt.PageSize
	pt.MapPage(vpn, seg.PhysicalPFN, true)

	smm.Mappings[shmID] = append(smm.Mappings[shmID], ProcessMapping{
		PID:            pid,
		VirtualAddress: vAddr,
		PageTable:      pt,
	})

	return nil
}

// WriteSync writes data to shared memory under protection of IPC semaphore
func (seg *SharedMemorySegment) WriteSync(data []byte) {
	seg.Semaphore.Wait()
	defer seg.Semaphore.Signal()

	copy(seg.PhysicalData, data)
}

// ReadSync reads data from shared memory under protection of IPC semaphore
func (seg *SharedMemorySegment) ReadSync() []byte {
	seg.Semaphore.Wait()
	defer seg.Semaphore.Signal()

	out := make([]byte, len(seg.PhysicalData))
	copy(out, seg.PhysicalData)
	return out
}

// RenderSharedMemoryDiagram visualizes shared physical frames mapped into different address spaces
func (smm *SharedMemoryManager) RenderSharedMemoryDiagram(shmID int) string {
	smm.mu.Lock()
	defer smm.mu.Unlock()

	var sb strings.Builder
	sb.WriteString(visualizer.SubHeader(fmt.Sprintf("Shared Memory IPC: Physical Frame Mappings (ShmID: %d)", shmID)))

	seg, exists := smm.Segments[shmID]
	if !exists {
		return "Segment not found\n"
	}

	mappings := smm.Mappings[shmID]

	tbl := visualizer.NewTable("Process", "Virtual Address", "VPN", "Mapped Hardware PFN", "Access Speed", "Payload")
	tbl.SetAlignment("center", "center", "center", "center", "center", "left")

	payloadStr := strings.TrimRight(string(seg.PhysicalData), "\x00")
	if payloadStr == "" {
		payloadStr = "(empty)"
	}

	for _, m := range mappings {
		tbl.AddRow(
			fmt.Sprintf("Process (PID %d)", m.PID),
			fmt.Sprintf("0x%08X", m.VirtualAddress),
			fmt.Sprintf("0x%04X", m.VirtualAddress/4096),
			visualizer.Green(fmt.Sprintf("PFN %d (Shared)", seg.PhysicalPFN)),
			visualizer.Cyan("Zero-Copy (Direct RAM)"),
			fmt.Sprintf("\"%s\"", payloadStr),
		)
	}

	sb.WriteString(tbl.Render())

	theory := []string{
		"HOW SHARED MEMORY ACHIEVES ZERO-COPY IPC:",
		"• Pipes & Sockets require 2 memory copies + 2 system calls:",
		"  - Process A (User Space) ──► sys_write (Copy 1) ──► Kernel Buffer ──► sys_read (Copy 2) ──► Process B (User Space)",
		"• Shared Memory requires ZERO memory copies and ZERO system calls during read/write:",
		"  - Both Process A and Process B have page table entries pointing to the EXACT same physical frame (PFN).",
		"  - Process A writes directly to memory bus; Process B sees it at hardware clock speed!",
		"• Synchronization Requirement:",
		"  - Because there is no kernel mediation on read/write, processes MUST synchronize using IPC Semaphores / Futexes to prevent race conditions.",
	}
	sb.WriteString("\n" + visualizer.Box("Zero-Copy Shared Memory Architecture", theory))

	return sb.String()
}
