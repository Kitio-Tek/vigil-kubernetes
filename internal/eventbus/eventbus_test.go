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

package eventbus_test

import (
	"sync/atomic"
	"testing"

	"github.com/Kitio-Tek/athos-kubernetes/internal/eventbus"
)

func TestBus_PublishDeliversToSubscribers(t *testing.T) {
	bus := eventbus.New()
	var count int32
	bus.Subscribe(eventbus.TopicClusterReady, func(_ eventbus.Event) {
		atomic.AddInt32(&count, 1)
	})
	bus.Publish(eventbus.Event{Topic: eventbus.TopicClusterReady})
	if got := atomic.LoadInt32(&count); got != 1 {
		t.Errorf("delivered %d times, want 1", got)
	}
}

func TestBus_PublishToOtherTopicIgnored(t *testing.T) {
	bus := eventbus.New()
	called := false
	bus.Subscribe(eventbus.TopicClusterReady, func(_ eventbus.Event) { called = true })
	bus.Publish(eventbus.Event{Topic: eventbus.TopicBackupCompleted})
	if called {
		t.Error("subscriber on different topic should not be called")
	}
}

func TestBus_MultipleSubscribers(t *testing.T) {
	bus := eventbus.New()
	var c1, c2 int32
	bus.Subscribe(eventbus.TopicBackupCompleted, func(_ eventbus.Event) { atomic.AddInt32(&c1, 1) })
	bus.Subscribe(eventbus.TopicBackupCompleted, func(_ eventbus.Event) { atomic.AddInt32(&c2, 1) })
	bus.Publish(eventbus.Event{Topic: eventbus.TopicBackupCompleted})
	if c1 != 1 || c2 != 1 {
		t.Errorf("counts = %d,%d", c1, c2)
	}
}

func TestBus_Unsubscribe(t *testing.T) {
	bus := eventbus.New()
	called := false
	unsub := bus.Subscribe(eventbus.TopicClusterReady, func(_ eventbus.Event) { called = true })
	unsub()
	bus.Publish(eventbus.Event{Topic: eventbus.TopicClusterReady})
	if called {
		t.Error("unsubscribed handler should not be called")
	}
}

func TestBus_TimeFilledIn(t *testing.T) {
	bus := eventbus.New()
	var got eventbus.Event
	bus.Subscribe(eventbus.TopicClusterReady, func(e eventbus.Event) { got = e })
	bus.Publish(eventbus.Event{Topic: eventbus.TopicClusterReady})
	if got.Time.IsZero() {
		t.Error("Time should be populated by Publish when zero")
	}
}

func TestBus_SubscriberCount(t *testing.T) {
	bus := eventbus.New()
	if got := bus.SubscriberCount(eventbus.TopicUserReconciled); got != 0 {
		t.Errorf("empty bus count = %d", got)
	}
	bus.Subscribe(eventbus.TopicUserReconciled, func(_ eventbus.Event) {})
	bus.Subscribe(eventbus.TopicUserReconciled, func(_ eventbus.Event) {})
	if got := bus.SubscriberCount(eventbus.TopicUserReconciled); got != 2 {
		t.Errorf("count = %d, want 2", got)
	}
}

func TestBus_Reset(t *testing.T) {
	bus := eventbus.New()
	bus.Subscribe(eventbus.TopicClusterReady, func(_ eventbus.Event) {})
	bus.Reset()
	if got := bus.SubscriberCount(eventbus.TopicClusterReady); got != 0 {
		t.Errorf("post-reset count = %d", got)
	}
}

func TestBus_ZeroValueWorks(t *testing.T) {
	var bus eventbus.Bus
	called := false
	bus.Subscribe(eventbus.TopicClusterReady, func(_ eventbus.Event) { called = true })
	bus.Publish(eventbus.Event{Topic: eventbus.TopicClusterReady})
	if !called {
		t.Error("zero-value bus should be usable")
	}
}
