package ipc

import (
	"errors"
	"fmt"
	"os-simulator/pkg/visualizer"
	"strings"
	"sync"
)

var (
	ErrQueueFull  = errors.New("message queue full")
	ErrQueueEmpty = errors.New("no message of requested type")
)

// IPCMessage represents a typed structured message in kernel message queue
type IPCMessage struct {
	MsgType int
	SenderPID int
	Payload string
}

// MessageQueue simulates POSIX/System V Message Queues
type MessageQueue struct {
	QueueID  int
	Capacity int
	Messages []IPCMessage
	mu       sync.Mutex
}

func NewMessageQueue(qid, capacity int) *MessageQueue {
	if capacity <= 0 {
		capacity = 10
	}
	return &MessageQueue{
		QueueID:  qid,
		Capacity: capacity,
		Messages: make([]IPCMessage, 0, capacity),
	}
}

// Send appends a typed message to the queue (msgsnd)
func (mq *MessageQueue) Send(msg IPCMessage) error {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	if len(mq.Messages) >= mq.Capacity {
		return ErrQueueFull
	}

	mq.Messages = append(mq.Messages, msg)
	return nil
}

// Receive retrieves the first message matching targetType (msgrcv). If targetType == 0, returns first message.
func (mq *MessageQueue) Receive(targetType int) (IPCMessage, error) {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	for i, msg := range mq.Messages {
		if targetType == 0 || msg.MsgType == targetType {
			mq.Messages = append(mq.Messages[:i], mq.Messages[i+1:]...)
			return msg, nil
		}
	}

	return IPCMessage{}, ErrQueueEmpty
}

// RenderQueueStatus visualizes the message queue contents
func (mq *MessageQueue) RenderQueueStatus() string {
	mq.mu.Lock()
	defer mq.mu.Unlock()

	var sb strings.Builder
	sb.WriteString(visualizer.SubHeader(fmt.Sprintf("Kernel Message Queue (Queue ID: %d, Capacity: %d)", mq.QueueID, mq.Capacity)))

	tbl := visualizer.NewTable("Index", "Msg Type", "Sender PID", "Message Payload")
	tbl.SetAlignment("center", "center", "center", "left")

	for i, msg := range mq.Messages {
		tbl.AddRow(
			fmt.Sprintf("#%d", i),
			fmt.Sprintf("Type %d", msg.MsgType),
			fmt.Sprintf("PID %d", msg.SenderPID),
			fmt.Sprintf("\"%s\"", msg.Payload),
		)
	}

	if len(mq.Messages) == 0 {
		tbl.AddRow("-", "-", "-", "(Queue is empty)")
	}

	sb.WriteString(tbl.Render())
	return sb.String()
}
