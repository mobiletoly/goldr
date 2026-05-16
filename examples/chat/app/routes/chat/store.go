package chat

import (
	"context"
	"slices"
	"sync"
	"time"
)

var room = newRoom()

type Message struct {
	ID        int64
	Author    string
	Body      string
	CreatedAt time.Time
}

type chatRoom struct {
	mu          sync.Mutex
	nextID      int64
	nextSubID   int64
	messages    []Message
	subscribers map[int64]chan Message
}

func newRoom() *chatRoom {
	return &chatRoom{
		subscribers: make(map[int64]chan Message),
	}
}

func listMessages() []Message {
	return room.messagesSnapshot()
}

func addMessage(author, body string) Message {
	return room.addMessage(author, body)
}

func subscribe(ctx context.Context, afterID int64) (<-chan Message, func()) {
	return room.subscribe(ctx, afterID)
}

func (room *chatRoom) messagesSnapshot() []Message {
	room.mu.Lock()
	defer room.mu.Unlock()
	return slices.Clone(room.messages)
}

func (room *chatRoom) addMessage(author, body string) Message {
	room.mu.Lock()
	room.nextID++
	message := Message{
		ID:        room.nextID,
		Author:    author,
		Body:      body,
		CreatedAt: time.Now(),
	}
	room.messages = append(room.messages, message)

	subscribers := make([]chan Message, 0, len(room.subscribers))
	for _, subscriber := range room.subscribers {
		subscribers = append(subscribers, subscriber)
	}
	room.mu.Unlock()

	for _, subscriber := range subscribers {
		select {
		case subscriber <- message:
		default:
		}
	}
	return message
}

func (room *chatRoom) subscribe(ctx context.Context, afterID int64) (<-chan Message, func()) {
	events := make(chan Message, 16)

	room.mu.Lock()
	room.nextSubID++
	subID := room.nextSubID
	room.subscribers[subID] = events
	replay := messagesAfter(room.messages, afterID)
	room.mu.Unlock()

	for _, message := range replay {
		events <- message
	}

	cancel := func() {
		room.mu.Lock()
		if subscriber, ok := room.subscribers[subID]; ok {
			delete(room.subscribers, subID)
			close(subscriber)
		}
		room.mu.Unlock()
	}

	go func() {
		<-ctx.Done()
		cancel()
	}()

	return events, cancel
}

func messagesAfter(messages []Message, afterID int64) []Message {
	if afterID <= 0 {
		return slices.Clone(messages)
	}
	var result []Message
	for _, message := range messages {
		if message.ID > afterID {
			result = append(result, message)
		}
	}
	return result
}
