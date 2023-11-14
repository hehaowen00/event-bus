package eventbus

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrEventBusClosed = errors.New("event bus closed")
)

type Topic[T any] struct {
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex

	history       History[T]
	incoming      chan T
	subscriptions []*Receiver[T]

	subscribeEvents   chan *Receiver[T]
	unsubscribeEvents chan *Receiver[T]
}

type Receiver[T any] struct {
	bus    *Topic[T]
	recv   chan T
	notify chan struct{}
	mu     sync.Mutex

	limit    int
	backfill bool
	queue    []T
}

func New[T any]() *Topic[T] {
	ctx, cancel := context.WithCancel(context.Background())

	bus := &Topic[T]{
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

func NewWithHistory[T any](strategy History[T]) *Topic[T] {
	ctx, cancel := context.WithCancel(context.Background())

	bus := &Topic[T]{
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

func (bus *Topic[T]) start() {
	for {
		select {
		case <-bus.ctx.Done():
			for i := range bus.subscriptions {
				r := bus.subscriptions[i]
				r.notify <- struct{}{}
			}
			break
		case client := <-bus.subscribeEvents:
			if client.backfill {
				go func() {
					client.mu.Lock()
					defer client.mu.Unlock()

					bus.history.fill(client.recv)
				}()
			}
			bus.subscriptions = append(bus.subscriptions, client)
		case _ = <-bus.unsubscribeEvents:
		case msg := <-bus.incoming:
			bus.history.append(msg)

			wait := make(chan struct{})

			go func() {
				wg := sync.WaitGroup{}

				for i := range bus.subscriptions {
					wg.Add(1)

					go func(i int) {
						defer wg.Done()
						r := bus.subscriptions[i]
						r.mu.Lock()
						r.queue = append(r.queue, msg)

						for len(r.queue) > 0 {
							if len(r.recv) > 0 {
								return
							}

							r.recv <- r.queue[0]
							r.queue = r.queue[1:]
						}
						r.mu.Unlock()
					}(i)
				}

				wg.Wait()

				close(wait)
			}()

			<-wait
		}
	}
}

func (bus *Topic[T]) Close() {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	bus.cancel()
}

func (bus *Topic[T]) Subscribe(backfill bool) (*Receiver[T], error) {
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

func (bus *Topic[T]) Unsubscribe(r *Receiver[T]) {
	bus.unsubscribeEvents <- r
}

func (bus *Topic[T]) Sender() chan<- T {
	return bus.incoming
}

func (bus *Topic[T]) Send(msg T) {
	bus.incoming <- msg
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
