package io

import (
	"testing"
	"time"
)

func TestBlockingIO(t *testing.T) {
	bio := NewBlockingIO()
	fd := &FileDescriptor{FD: 1, Name: "socket-1", DataBuffer: []byte("hello")}

	data, elapsed := bio.Read(fd, 5*time.Millisecond)
	if string(data) != "hello" {
		t.Errorf("Expected 'hello', got '%s'", string(data))
	}
	if elapsed < 5*time.Millisecond {
		t.Errorf("Expected elapsed >= 5ms, got %v", elapsed)
	}
}

func TestSelectPoll(t *testing.T) {
	sp := NewSelectPoll()
	fds := []*FileDescriptor{
		{FD: 1, IsReady: false},
		{FD: 2, IsReady: true},
		{FD: 3, IsReady: false},
	}

	ready, count, _ := sp.Poll(fds)
	if len(ready) != 1 || ready[0].FD != 2 {
		t.Errorf("Expected ready FD 2, got %+v", ready)
	}
	if count != 3 {
		t.Errorf("Expected scanned count 3, got %d", count)
	}
}

func TestEpollReadyNotification(t *testing.T) {
	ep := NewEpoll()
	fd1 := &FileDescriptor{FD: 10}
	fd2 := &FileDescriptor{FD: 20}

	ep.EpollCtl(EPOLL_CTL_ADD, fd1, &EpollEvent{Events: EPOLLIN, FD: fd1})
	ep.EpollCtl(EPOLL_CTL_ADD, fd2, &EpollEvent{Events: EPOLLIN, FD: fd2})

	// Trigger interrupt on fd2
	ep.NotifyDeviceInterrupt(fd2, EPOLLIN)

	events, err := ep.EpollWait(10, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("EpollWait failed: %v", err)
	}

	if len(events) != 1 || events[0].FD.FD != 20 {
		t.Fatalf("Expected 1 event on FD 20, got %+v", events)
	}
}

func TestGoNetpoller(t *testing.T) {
	np := NewGoNetpoller()
	fd := &FileDescriptor{FD: 100}

	// 1. Read on unready socket -> G parked
	_, ready := np.ReadNonBlocking(1, fd)
	if ready {
		t.Fatalf("Expected G to be parked")
	}

	// 2. Simulate socket data arrival via interrupt
	fd.DataBuffer = []byte("response-packet")
	np.EpollInstance.NotifyDeviceInterrupt(fd, EPOLLIN)

	// 3. Sysmon netpoll
	woken := np.PollAndWakeup()
	if len(woken) != 1 || woken[0].GID != 1 {
		t.Fatalf("Expected G1 to be woken, got %+v", woken)
	}
}
