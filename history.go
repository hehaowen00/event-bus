package eventbus

import (
	"sync"
)

type History[T any] interface {
	Append(T)
	Data() []T
}

type EmptyHistory[T any] struct {
}

func NewEmptyHistory[T any]() History[T] {
	return &EmptyHistory[T]{}
}

func (hist *EmptyHistory[T]) Append(_ T) {}

func (hist *EmptyHistory[T]) Data() []T {
	return []T{}
}

type FixedHistory[T any] struct {
	data  []T
	size  int
	index int
	mu    sync.Mutex
}

func NewFixedHistory[T any](size int) History[T] {
	return &FixedHistory[T]{
		data: make([]T, size),
		size: size,
	}
}

func (hist *FixedHistory[T]) Append(msg T) {
	hist.mu.Lock()
	hist.data[hist.index] = msg
	hist.index = (hist.index + 1) % hist.size
	hist.mu.Unlock()
}

func (hist *FixedHistory[T]) Data() []T {
	return hist.data
}

type UnboundedHistory[T any] struct {
	data []T
	mu   sync.Mutex
}

func NewUnboundedHistory[T any]() History[T] {
	return &UnboundedHistory[T]{}
}

func (hist *UnboundedHistory[T]) Append(msg T) {
	hist.mu.Lock()
	hist.data = append(hist.data, msg)
	hist.mu.Unlock()
}

func (hist *UnboundedHistory[T]) Data() []T {
	return hist.data
}
