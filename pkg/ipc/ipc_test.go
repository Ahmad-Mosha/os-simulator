package ipc

import (
	"os-simulator/pkg/memory"
	"sync"
	"testing"
)

func TestUnixPipeReadWriteEOF(t *testing.T) {
	pipe := NewUnixPipe(64)

	var wg sync.WaitGroup
	var readData []byte

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		pipe.Write([]byte("Hello through Unix Pipe!"))
		pipe.CloseWrite() // Send EOF
	}()

	// Reader goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 128)
		n, err := pipe.Read(buf)
		if err != nil {
			t.Errorf("Pipe read failed: %v", err)
			return
		}
		readData = buf[:n]
	}()

	wg.Wait()

	if string(readData) != "Hello through Unix Pipe!" {
		t.Errorf("Expected 'Hello through Unix Pipe!', got '%s'", string(readData))
	}

	// Verify reading from closed empty pipe returns 0 bytes (EOF)
	buf := make([]byte, 10)
	n, err := pipe.Read(buf)
	if err != nil || n != 0 {
		t.Errorf("Expected EOF (0 bytes, nil err), got n=%d, err=%v", n, err)
	}
}

func TestSharedMemoryZeroCopy(t *testing.T) {
	smm := NewSharedMemoryManager()
	seg, err := smm.CreateSegment(64)
	if err != nil {
		t.Fatalf("CreateSegment failed: %v", err)
	}

	// Process 1 (PID 10) attaches at VA 0x00500000
	pt1 := memory.NewPageTable(10, 1024, 4096)
	smm.Attach(seg.ShmID, 10, 0x00500000, pt1)

	// Process 2 (PID 20) attaches at DIFFERENT VA 0x00700000
	pt2 := memory.NewPageTable(20, 1024, 4096)
	smm.Attach(seg.ShmID, 20, 0x00700000, pt2)

	// Process 1 writes data
	seg.WriteSync([]byte("Shared Payload 123"))

	// Process 2 reads data
	data := seg.ReadSync()
	if string(data[:18]) != "Shared Payload 123" {
		t.Errorf("Expected 'Shared Payload 123', got '%s'", string(data))
	}
}

func TestMessageQueue(t *testing.T) {
	mq := NewMessageQueue(1, 5)

	mq.Send(IPCMessage{MsgType: 1, SenderPID: 100, Payload: "High Priority Alert"})
	mq.Send(IPCMessage{MsgType: 2, SenderPID: 101, Payload: "Regular Log"})

	// Receive specific MsgType 2
	msg2, err := mq.Receive(2)
	if err != nil || msg2.Payload != "Regular Log" {
		t.Errorf("Expected MsgType 2 'Regular Log', got %+v, err=%v", msg2, err)
	}

	// Receive remaining MsgType 1
	msg1, err := mq.Receive(0)
	if err != nil || msg1.Payload != "High Priority Alert" {
		t.Errorf("Expected MsgType 1 'High Priority Alert', got %+v, err=%v", msg1, err)
	}
}
