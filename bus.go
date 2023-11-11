package eventbus

import (
	"context"
	"errors"
)

var (
	ErrEventBusClosed = errors.New("event bus closed")
)

type EventBus[T any] struct {
	ctx    context.Context
	cancel context.CancelFunc

	backlog       []T
	limit         int
	incoming      chan T
	subscriptions []*Receiver[T]

	subscribeEvents   chan *Receiver[T]
	unsubscribeEvents chan *Receiver[T]
}

type Receiver[T any] struct {
	bus    *EventBus[T]
	recv   chan T
	notify chan struct{}

	limit    int
	backfill bool
}

func New[T any](backlog int) *EventBus[T] {
	ctx, cancel := context.WithCancel(context.Background())

	bus := &EventBus[T]{
		ctx:    ctx,
		cancel: cancel,

		limit:             backlog,
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
					for i := range bus.backlog {
						client.recv <- bus.backlog[i]
					}
				}()
			}
		case _ = <-bus.unsubscribeEvents:
		case msg := <-bus.incoming:
			bus.backlog = append(bus.backlog, msg)

			for i := range bus.subscriptions {
				bus.subscriptions[i].recv <- msg
			}

			if len(bus.backlog) > bus.limit {
				bus.backlog = bus.backlog[1:]
			}
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
			recv:     make(chan T, bus.limit),
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
