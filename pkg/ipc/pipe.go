package ipc

import (
	"errors"
	"fmt"
	"os-simulator/pkg/visualizer"
	"strings"
	"sync"
)

var (
	ErrPipeClosed = errors.New("broken pipe (EPIPE): write to pipe with no readers (SIGPIPE)")
	ErrPipeEmpty  = errors.New("pipe empty (EAGAIN)")
	ErrPipeFull   = errors.New("pipe full: buffer capacity reached")
)

// UnixPipe simulates an anonymous kernel IPC pipe
// Structure: In-kernel circular ring buffer with read and write file descriptors
type UnixPipe struct {
	Capacity     int
	Buffer       []byte
	ReadClosed   bool
	WriteClosed  bool
	TotalWritten int
	TotalRead    int
	mu           sync.Mutex
	notEmpty     *sync.Cond
	notFull      *sync.Cond
}

func NewUnixPipe(capacity int) *UnixPipe {
	if capacity <= 0 {
		capacity = 4096 // 4 KB standard pipe buffer size
	}
	p := &UnixPipe{
		Capacity: capacity,
		Buffer:   make([]byte, 0, capacity),
	}
	p.notEmpty = sync.NewCond(&p.mu)
	p.notFull = sync.NewCond(&p.mu)
	return p
}

// Write writes bytes into the kernel pipe buffer (blocks if buffer is full)
func (p *UnixPipe) Write(data []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Writing to a pipe whose read end is closed generates SIGPIPE
	if p.ReadClosed {
		return 0, ErrPipeClosed
	}

	written := 0
	for _, b := range data {
		for len(p.Buffer) >= p.Capacity && !p.ReadClosed {
			p.notFull.Wait()
		}
		if p.ReadClosed {
			return written, ErrPipeClosed
		}

		p.Buffer = append(p.Buffer, b)
		written++
		p.TotalWritten++
		p.notEmpty.Signal() // Signal waiting reader
	}

	return written, nil
}

// Read reads bytes from the kernel pipe buffer into dst (blocks if empty until data or EOF)
func (p *UnixPipe) Read(dst []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Wait while buffer is empty and writer is still open
	for len(p.Buffer) == 0 && !p.WriteClosed {
		p.notEmpty.Wait()
	}

	// EOF: Buffer empty and write end closed
	if len(p.Buffer) == 0 && p.WriteClosed {
		return 0, nil // Standard Unix EOF (0 bytes read)
	}

	toRead := len(dst)
	if toRead > len(p.Buffer) {
		toRead = len(p.Buffer)
	}

	copy(dst[:toRead], p.Buffer[:toRead])
	p.Buffer = p.Buffer[toRead:]
	p.TotalRead += toRead
	p.notFull.Signal() // Signal waiting writer

	return toRead, nil
}

// CloseWrite closes the write end of the pipe (signals EOF to reader)
func (p *UnixPipe) CloseWrite() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.WriteClosed = true
	p.notEmpty.Broadcast() // Wake up blocked readers so they see EOF
}

// CloseRead closes the read end of the pipe (subsequent writes trigger SIGPIPE / EPIPE)
func (p *UnixPipe) CloseRead() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.ReadClosed = true
	p.notFull.Broadcast() // Wake up blocked writers so they receive EPIPE
}

// RenderPipeStatus visualizes the in-kernel pipe buffer
func (p *UnixPipe) RenderPipeStatus() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	var sb strings.Builder
	sb.WriteString(visualizer.SubHeader("Unix Anonymous Pipe IPC (Kernel Ring Buffer)"))

	tbl := visualizer.NewTable("Buffer Capacity", "Buffered Bytes", "Total Written", "Total Read", "Read End", "Write End")
	tbl.SetAlignment("center", "center", "center", "center", "center", "center")

	rState := visualizer.Badge("OPEN (fd[0])", visualizer.BgGreen, visualizer.FgHiWhite)
	if p.ReadClosed {
		rState = visualizer.Badge("CLOSED", visualizer.BgRed, visualizer.FgHiWhite)
	}

	wState := visualizer.Badge("OPEN (fd[1])", visualizer.BgGreen, visualizer.FgHiWhite)
	if p.WriteClosed {
		wState = visualizer.Badge("CLOSED (EOF)", visualizer.BgYellow, visualizer.FgHiWhite)
	}

	tbl.AddRow(
		fmt.Sprintf("%d bytes", p.Capacity),
		fmt.Sprintf("%d bytes", len(p.Buffer)),
		fmt.Sprintf("%d bytes", p.TotalWritten),
		fmt.Sprintf("%d bytes", p.TotalRead),
		rState,
		wState,
	)

	sb.WriteString(tbl.Render())

	pipeInfo := []string{
		"UNIX PIPE ARCHITECTURE (OSTEP Chapter 5):",
		"• Unidirectional Byte Stream: Created via 'pipe(int fds[2])'.",
		"  - fds[0] = Read end",
		"  - fds[1] = Write end",
		"• In-Kernel Ring Buffer: Data resides in kernel RAM without ever touching physical disk.",
		"• Synchronization Semantics:",
		"  - Read blocks automatically when pipe is empty.",
		"  - Write blocks automatically when pipe is full (Flow Control / Backpressure).",
		"  - Closing write end causes reader to receive EOF (0 bytes).",
		"  - Writing to a pipe with closed read end raises SIGPIPE / -EPIPE.",
	}
	sb.WriteString("\n" + visualizer.Box("Pipe IPC Internals & Semantics", pipeInfo))

	return sb.String()
}
