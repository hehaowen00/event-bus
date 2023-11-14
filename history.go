package eventbus

import (
	"sync"
)

type History[T any] interface {
	Prefill([]T)
	append(T)
	fill(chan<- T)
}

type EmptyHistory[T any] struct {
}

func NewEmptyHistory[T any]() History[T] {
	return &EmptyHistory[T]{}
}

func (hist *EmptyHistory[T]) Prefill(data []T) {}

func (hist *EmptyHistory[T]) append(_ T) {}

func (hist *EmptyHistory[T]) fill(_ chan<- T) {}

type FixedHistory[T any] struct {
	data  []T
	size  int
	index int
	mu    sync.Mutex
}

func NewFixedHistory[T any](size int) History[T] {
	return &FixedHistory[T]{
		data: make([]T, size, size),
		size: size,
	}
}

func (hist *FixedHistory[T]) Prefill(data []T) {
	for i := range data {
		hist.append(data[i])
	}
}

func (hist *FixedHistory[T]) append(msg T) {
	hist.mu.Lock()
	defer hist.mu.Unlock()

	hist.data[hist.index] = msg
	hist.index = (hist.index + 1) % hist.size
}

func (hist *FixedHistory[T]) fill(rx chan<- T) {
	hist.mu.Lock()
	defer hist.mu.Unlock()

	go func() {
		clone := hist
		for i := 0; i < hist.size; i++ {
			rx <- clone.data[i]
		}
	}()
}

type UnboundedHistory[T any] struct {
	data []T
	mu   sync.Mutex
}

func NewUnboundedHistory[T any]() History[T] {
	return &UnboundedHistory[T]{}
}

func (hist *UnboundedHistory[T]) Prefill(data []T) {
	hist.data = data
}

func (hist *UnboundedHistory[T]) append(msg T) {
	hist.mu.Lock()
	defer hist.mu.Unlock()

	hist.data = append(hist.data, msg)
}

func (hist *UnboundedHistory[T]) fill(rx chan<- T) {
	hist.mu.Lock()
	defer hist.mu.Unlock()

	go func() {
		clone := hist.data
		for i := range clone {
			rx <- clone[i]
		}
	}()
}
