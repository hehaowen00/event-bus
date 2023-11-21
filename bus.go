package eventbus

import (
	"sync"
)

type EventBus[T any] struct {
	topics map[string]*Topic[T]
	mu     sync.Mutex
}

func NewEventBus[T any]() *EventBus[T] {
	bus := EventBus[T]{
		topics: make(map[string]*Topic[T]),
	}
	return &bus
}

func (bus *EventBus[T]) Add(name string, topic *Topic[T]) {
	bus.mu.Lock()

	existing, ok := bus.topics[name]
	if !ok {
		bus.topics[name] = topic
		return
	}

	existing.Close()
	bus.topics[name] = topic

	bus.mu.Unlock()
}

func (bus *EventBus[T]) Get(name string) *Topic[T] {
	bus.mu.Lock()
	topic, ok := bus.topics[name]
	bus.mu.Unlock()

	if !ok {
		return nil
	}

	return topic
}

func (bus *EventBus[T]) Close() {
	bus.mu.Lock()

	wg := sync.WaitGroup{}

	for _, topic := range bus.topics {
		wg.Add(1)
		go func(t *Topic[T]) {
			t.Close()
			wg.Done()
		}(topic)
	}

	wg.Wait()

	bus.mu.Unlock()
}

func (bus *EventBus[T]) Remove(name string) {
	bus.topics[name].Close()

	bus.mu.Lock()
	delete(bus.topics, name)
	bus.mu.Unlock()
}
