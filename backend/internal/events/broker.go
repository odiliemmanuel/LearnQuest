package events

import (
	"context"
	"sync"
	"time"
)

const (
	QuizCompleted = "quiz.completed"
)

type Event struct {
	Type    string
	Data    any
	TraceID string
}

type Handler func(ctx context.Context, e Event) error

type Broker struct {
	mu     sync.RWMutex
	subs   map[string][]Handler
	wg     sync.WaitGroup
	sem    chan struct{}
	closed bool
}

func NewBroker(maxConcurrent int) *Broker {
	if maxConcurrent <= 0 {
		maxConcurrent = 8
	}
	return &Broker{subs: map[string][]Handler{}, sem: make(chan struct{}, maxConcurrent)}
}

func (b *Broker) Subscribe(typ string, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.subs[typ] = append(b.subs[typ], h)
}

func (b *Broker) Publish(typ string, data any, traceID string) {
	b.mu.RLock()
	handlers := append([]Handler(nil), b.subs[typ]...)
	b.mu.RUnlock()
	if len(handlers) == 0 {
		return
	}
	e := Event{Type: typ, Data: data, TraceID: traceID}
	for _, h := range handlers {
		b.wg.Add(1)
		b.sem <- struct{}{}
		go func(h Handler) {
			defer b.wg.Done()
			defer func() { <-b.sem }()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			_ = h(ctx, e)
		}(h)
	}
}

func (b *Broker) Shutdown(ctx context.Context) {
	b.mu.Lock()
	b.closed = true
	b.mu.Unlock()
	done := make(chan struct{})
	go func() { b.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
	}
}
