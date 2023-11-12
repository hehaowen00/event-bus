package eventbus

import "sync"

type HistoryStrategy[T any] interface {
	append(T)
	fill(chan<- T)
}

type EmptyHistory[T any] struct {
}

func NewEmptyHistory[T any]() HistoryStrategy[T] {
	return &EmptyHistory[T]{}
}

func (hist *EmptyHistory[T]) append(_ T) {}

func (hist *EmptyHistory[T]) fill(_ chan<- T) {}

type FixedHistory[T any] struct {
	data  []T
	count int
	mu    sync.Mutex
}

func NewFixedHistory[T any](count int) HistoryStrategy[T] {
	return &FixedHistory[T]{
		count: count,
	}
}

func (hist *FixedHistory[T]) Prefill(data []T) {
	hist.data = data
}

func (hist *FixedHistory[T]) append(msg T) {
	hist.mu.Lock()
	defer hist.mu.Unlock()

	hist.data = append(hist.data, msg)
	if len(hist.data) > hist.count {
		hist.data = hist.data[1:]
	}
}

func (hist *FixedHistory[T]) fill(rx chan<- T) {
	for i := range hist.data {
		rx <- hist.data[i]
	}
}

type UnboundedHistory[T any] struct {
	data []T
	mu   sync.Mutex
}

func NewUnboundedHistory[T any]() HistoryStrategy[T] {
	return &UnboundedHistory[T]{}
}

func (hist *UnboundedHistory[T]) append(msg T) {
	hist.mu.Lock()
	defer hist.mu.Unlock()

	hist.data = append(hist.data, msg)
}

func (hist *UnboundedHistory[T]) fill(rx chan<- T) {
	hist.mu.Lock()
	defer hist.mu.Unlock()

	for i := range hist.data {
		rx <- hist.data[i]
	}
}
