// Package eventbus is the in-process notification bus (AIA-03 §10).
//
// The bus is a notification mechanism only; it is never the source of truth.
// Subscribers that miss events (buffer overflow or replay gap) must re-query
// state through the regular services / REST API.
package eventbus

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Event types published by the runtime.
const (
	TaskCreated         = "task.created"
	TaskUpdated         = "task.updated"
	TaskCompleted       = "task.completed"
	TaskFailed          = "task.failed"
	TaskWaitingApproval = "task.waiting_approval"
	TaskWaitingAuth     = "task.waiting_auth"
	AgentStatusChanged  = "agent.status_changed"
	SystemHealthChanged = "system.health_changed"
	InvestmentUpdated   = "investment.updated"
)

// Event is a single notification.
type Event struct {
	Seq  uint64          `json:"-"`
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
	Time time.Time       `json:"time"`
}

// Publisher publishes events.
type Publisher interface {
	Publish(eventType string, data any)
}

// Subscription is a live event subscription.
type Subscription struct {
	C      <-chan Event
	cancel func()
	// Dropped reports whether events were dropped because the subscriber was
	// too slow; the consumer should resync via REST.
	dropped *bool
	mu      *sync.Mutex
}

// Close stops the subscription.
func (s *Subscription) Close() { s.cancel() }

// Dropped reports whether any event was dropped for this subscriber.
func (s *Subscription) Dropped() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return *s.dropped
}

type subscriber struct {
	ch      chan Event
	dropped bool
}

// Bus is a bounded, replayable in-memory event bus.
type Bus struct {
	mu       sync.Mutex
	seq      uint64
	ring     []Event
	ringSize int
	subs     map[*subscriber]struct{}
	closed   bool
}

// New creates a bus that retains the last ringSize events for replay.
func New(ringSize int) *Bus {
	if ringSize <= 0 {
		ringSize = 1000
	}
	return &Bus{ringSize: ringSize, subs: make(map[*subscriber]struct{})}
}

// FormatID renders a sequence as an event ID.
func FormatID(seq uint64) string { return "evt_" + strconv.FormatUint(seq, 10) }

// ParseID parses an event ID produced by FormatID.
func ParseID(id string) (uint64, bool) {
	id = strings.TrimSpace(id)
	if !strings.HasPrefix(id, "evt_") {
		return 0, false
	}
	n, err := strconv.ParseUint(strings.TrimPrefix(id, "evt_"), 10, 64)
	return n, err == nil
}

// Publish broadcasts an event. It never blocks: slow subscribers lose events
// and are flagged so they can resync.
func (b *Bus) Publish(eventType string, data any) {
	raw, err := json.Marshal(data)
	if err != nil {
		raw = json.RawMessage("{}")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.seq++
	ev := Event{Seq: b.seq, ID: FormatID(b.seq), Type: eventType, Data: raw, Time: time.Now().UTC()}
	b.ring = append(b.ring, ev)
	if len(b.ring) > b.ringSize {
		b.ring = b.ring[len(b.ring)-b.ringSize:]
	}
	for s := range b.subs {
		select {
		case s.ch <- ev:
		default:
			s.dropped = true
		}
	}
}

// Subscribe registers a subscriber. If lastID is non-empty, events after it are
// returned for replay. complete is false when the requested history is no
// longer available (gap) and the caller must resync from REST.
func (b *Bus) Subscribe(lastID string, buffer int) (sub *Subscription, replay []Event, complete bool) {
	if buffer <= 0 {
		buffer = 64
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	complete = true
	if lastID != "" {
		last, ok := ParseID(lastID)
		switch {
		case !ok || last > b.seq:
			complete = false
		case last < b.seq:
			oldest := b.seq + 1
			if len(b.ring) > 0 {
				oldest = b.ring[0].Seq
			}
			if last+1 < oldest {
				complete = false
			}
			for _, ev := range b.ring {
				if ev.Seq > last {
					replay = append(replay, ev)
				}
			}
		}
	}

	s := &subscriber{ch: make(chan Event, buffer)}
	if !b.closed {
		b.subs[s] = struct{}{}
	} else {
		close(s.ch)
	}
	var once sync.Once
	sub = &Subscription{
		C:       s.ch,
		dropped: &s.dropped,
		mu:      &b.mu,
		cancel: func() {
			once.Do(func() {
				b.mu.Lock()
				defer b.mu.Unlock()
				if _, ok := b.subs[s]; ok {
					delete(b.subs, s)
					close(s.ch)
				}
			})
		},
	}
	return sub, replay, complete
}

// LastID returns the ID of the most recent event, or "" if none.
func (b *Bus) LastID() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.seq == 0 {
		return ""
	}
	return FormatID(b.seq)
}

// SubscriberCount returns the number of active subscribers.
func (b *Bus) SubscriberCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.subs)
}

// Close closes all subscriptions; further publishes are ignored.
func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	for s := range b.subs {
		delete(b.subs, s)
		close(s.ch)
	}
}
