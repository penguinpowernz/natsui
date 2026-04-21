package main

import (
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

type Message struct {
	Subject   string
	Payload   string
	Timestamp time.Time
}

type Subscription struct {
	Subject      string
	NatsSub      *nats.Subscription
	Messages     []*Message
	MessageCount int
}

type NATSManager struct {
	conn          *nats.Conn
	subscriptions map[string]*Subscription
	mu            sync.RWMutex
	onUpdate      func()
}

func NewNATSManager(url string, onUpdate func()) (*NATSManager, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}

	return &NATSManager{
		conn:          conn,
		subscriptions: make(map[string]*Subscription),
		onUpdate:      onUpdate,
	}, nil
}

func (nm *NATSManager) Subscribe(subject string) error {
	nm.mu.Lock()

	if _, exists := nm.subscriptions[subject]; exists {
		nm.mu.Unlock()
		return nil
	}

	sub, err := nm.conn.Subscribe(subject, func(msg *nats.Msg) {
		nm.handleMessage(subject, msg)
	})
	if err != nil {
		nm.mu.Unlock()
		return err
	}

	nm.subscriptions[subject] = &Subscription{
		Subject:  subject,
		NatsSub:  sub,
		Messages: make([]*Message, 0),
	}

	log.Printf("Subscribed to: %s", subject)
	nm.mu.Unlock()

	if nm.onUpdate != nil {
		nm.onUpdate()
	}

	return nil
}

func (nm *NATSManager) Unsubscribe(subject string) error {
	nm.mu.Lock()

	sub, exists := nm.subscriptions[subject]
	if !exists {
		nm.mu.Unlock()
		return nil
	}

	if err := sub.NatsSub.Unsubscribe(); err != nil {
		nm.mu.Unlock()
		return err
	}

	delete(nm.subscriptions, subject)
	log.Printf("Unsubscribed from: %s", subject)
	nm.mu.Unlock()

	if nm.onUpdate != nil {
		nm.onUpdate()
	}

	return nil
}

func (nm *NATSManager) ClearMessages(subject string) {
	nm.mu.Lock()

	sub, exists := nm.subscriptions[subject]
	if !exists {
		nm.mu.Unlock()
		return
	}

	sub.Messages = make([]*Message, 0)
	sub.MessageCount = 0
	log.Printf("Cleared messages for: %s", subject)
	nm.mu.Unlock()

	if nm.onUpdate != nil {
		nm.onUpdate()
	}
}

func (nm *NATSManager) GetSubscriptions() []string {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	subjects := make([]string, 0, len(nm.subscriptions))
	for subject := range nm.subscriptions {
		subjects = append(subjects, subject)
	}
	return subjects
}

func (nm *NATSManager) GetMessages(subject string) []*Message {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	if sub, exists := nm.subscriptions[subject]; exists {
		return sub.Messages
	}
	return nil
}

func (nm *NATSManager) GetMessageCount(subject string) int {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	if sub, exists := nm.subscriptions[subject]; exists {
		return sub.MessageCount
	}
	return 0
}

func (nm *NATSManager) handleMessage(subject string, msg *nats.Msg) {
	nm.mu.Lock()

	sub, exists := nm.subscriptions[subject]
	if !exists {
		nm.mu.Unlock()
		return
	}

	message := &Message{
		Subject:   msg.Subject,
		Payload:   string(msg.Data),
		Timestamp: time.Now(),
	}

	sub.Messages = append(sub.Messages, message)
	sub.MessageCount++

	// Keep only last 1000 messages
	if len(sub.Messages) > 1000 {
		sub.Messages = sub.Messages[len(sub.Messages)-1000:]
	}

	log.Printf("Received message on %s: %d bytes", subject, len(msg.Data))
	nm.mu.Unlock()

	if nm.onUpdate != nil {
		nm.onUpdate()
	}
}

func (nm *NATSManager) Close() {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	for _, sub := range nm.subscriptions {
		sub.NatsSub.Unsubscribe()
	}
	nm.conn.Close()
}
