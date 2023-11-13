package eventbus

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrEventBusClosed = errors.New("event bus closed")
)

type EventBus[T any] struct {
	ctx    context.Context
	cancel context.CancelFunc

	history       HistoryStrategy[T]
	incoming      chan T
	subscriptions []*Receiver[T]

	subscribeEvents   chan *Receiver[T]
	unsubscribeEvents chan *Receiver[T]
}

type Receiver[T any] struct {
	bus    *EventBus[T]
	recv   chan T
	notify chan struct{}
	mu     sync.Mutex

	limit    int
	backfill bool
	queue    []T
}

func New[T any]() *EventBus[T] {
	ctx, cancel := context.WithCancel(context.Background())

	bus := &EventBus[T]{
		ctx:    ctx,
		cancel: cancel,

		history:           NewEmptyHistory[T](),
		incoming:          make(chan T),
		subscribeEvents:   make(chan *Receiver[T]),
		unsubscribeEvents: make(chan *Receiver[T]),
	}

	go bus.start()

	return bus
}

func NewWithHistory[T any](strategy HistoryStrategy[T]) *EventBus[T] {
	ctx, cancel := context.WithCancel(context.Background())

	bus := &EventBus[T]{
		ctx:    ctx,
		cancel: cancel,

		history:           strategy,
		incoming:          make(chan T),
		subscribeEvents:   make(chan *Receiver[T]),
		unsubscribeEvents: make(chan *Receiver[T]),
	}

	go bus.start()

	return bus
}

func (bus *EventBus[T]) start() {
	for {
		select {
		case <-bus.ctx.Done():
			for i := range bus.subscriptions {
				bus.subscriptions[i].notify <- struct{}{}
			}
			return
		case client := <-bus.subscribeEvents:
			bus.subscriptions = append(bus.subscriptions, client)
			if client.backfill {
				go func() {
					bus.history.fill(client.recv)
				}()
			}
		case _ = <-bus.unsubscribeEvents:
		case msg := <-bus.incoming:
			bus.history.append(msg)

			wg := sync.WaitGroup{}

			for i := range bus.subscriptions {
				go func(i int) {
					wg.Add(1)
					defer wg.Done()

					r := bus.subscriptions[i]
					r.mu.Lock()
					defer r.mu.Unlock()

					r.queue = append(r.queue, msg)

					for len(r.queue) > 0 {
						r.recv <- r.queue[0]
						r.queue = r.queue[1:]
					}
				}(i)
			}

			wg.Wait()
		}
	}
}

func (bus *EventBus[T]) Close() {
	bus.cancel()
}

func (bus *EventBus[T]) Subscribe(backfill bool) (*Receiver[T], error) {
	select {
	case <-bus.ctx.Done():
		return nil, ErrEventBusClosed
	default:
		rx := &Receiver[T]{
			bus:      bus,
			recv:     make(chan T, 1),
			notify:   make(chan struct{}),
			backfill: backfill,
		}

		bus.subscribeEvents <- rx

		return rx, nil
	}
}

func (bus *EventBus[T]) Unsubscribe(r *Receiver[T]) {
	bus.unsubscribeEvents <- r
}

func (bus *EventBus[T]) Send() chan<- T {
	return bus.incoming
}

func (rx *Receiver[T]) Notify() <-chan struct{} {
	return rx.notify
}

func (rx *Receiver[T]) Recv() <-chan T {
	return rx.recv
}

func (rx *Receiver[T]) Close() {
	rx.bus.Unsubscribe(rx)
}

func (rx *Receiver[T]) Queue() []T {
	return rx.queue
}
