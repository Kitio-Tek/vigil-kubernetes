/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package eventbus provides a tiny in-process pub/sub used by the controllers
// to broadcast lifecycle hints to other reconcilers. It is intentionally
// minimal: there is no buffering, no persistence, and subscribers run in the
// publisher's goroutine.
package eventbus

import (
	"sync"
	"time"
)

// Topic is a string identifier for an event channel.
type Topic string

// Standard topics. Reconcilers should depend on these constants rather than
// hard-coded strings.
const (
	// TopicClusterReady fires when a PostgresCluster transitions to Ready.
	TopicClusterReady Topic = "cluster.ready"
	// TopicClusterDegraded fires when a PostgresCluster becomes degraded.
	TopicClusterDegraded Topic = "cluster.degraded"
	// TopicBackupCompleted fires when a PostgresBackup reaches Completed.
	TopicBackupCompleted Topic = "backup.completed"
	// TopicBackupFailed fires when a PostgresBackup reaches Failed.
	TopicBackupFailed Topic = "backup.failed"
	// TopicUserReconciled fires after a successful PostgresUser reconcile.
	TopicUserReconciled Topic = "user.reconciled"
)

// Event is the payload published on a Topic. Cluster and Namespace identify
// the originating PostgresCluster; Payload may carry topic-specific fields.
type Event struct {
	Topic     Topic
	Cluster   string
	Namespace string
	Time      time.Time
	Payload   map[string]string
}

// Handler is invoked synchronously when a matching event is published.
type Handler func(Event)

// Bus is an in-process pub/sub. The zero value is ready to use.
type Bus struct {
	mu       sync.RWMutex
	handlers map[Topic][]Handler
}

// New returns an empty bus. Callers may also use the zero value.
func New() *Bus { return &Bus{} }

// Subscribe registers handler for the given topic and returns a function that
// unsubscribes when called.
func (b *Bus) Subscribe(topic Topic, handler Handler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.handlers == nil {
		b.handlers = map[Topic][]Handler{}
	}
	b.handlers[topic] = append(b.handlers[topic], handler)
	idx := len(b.handlers[topic]) - 1
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		hs := b.handlers[topic]
		if idx >= len(hs) {
			return
		}
		hs[idx] = nil
		out := hs[:0]
		for _, h := range hs {
			if h != nil {
				out = append(out, h)
			}
		}
		b.handlers[topic] = out
	}
}

// Publish synchronously delivers e to every handler subscribed to its topic.
// If e.Time is zero, it is set to time.Now() before delivery.
func (b *Bus) Publish(e Event) {
	if e.Time.IsZero() {
		e.Time = time.Now()
	}
	b.mu.RLock()
	handlers := append([]Handler(nil), b.handlers[e.Topic]...)
	b.mu.RUnlock()
	for _, h := range handlers {
		h(e)
	}
}

// SubscriberCount returns how many handlers are currently registered for
// the given topic. Useful for diagnostics and tests.
func (b *Bus) SubscriberCount(topic Topic) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.handlers[topic])
}

// Reset removes every subscriber. Intended for use in tests.
func (b *Bus) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = nil
}
