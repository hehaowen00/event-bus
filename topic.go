package eventbus

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrTopicClosed = errors.New("topic closed")
)

type Topic[T any] struct {
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex

	history       History[T]
	incoming      chan T
	subscriptions []*Receiver[T]

	subscribeEvents   chan *subscription[T]
	unsubscribeEvents chan *Receiver[T]
}

type Receiver[T any] struct {
	topic  *Topic[T]
	recv   chan struct{}
	notify chan struct{}
	mu     sync.Mutex
	queue  []T
}

type subscription[T any] struct {
	rx       *Receiver[T]
	backfill bool
}

func New[T any]() *Topic[T] {
	return NewWithHistory[T](NewEmptyHistory[T]())
}

func NewWithHistory[T any](strategy History[T]) *Topic[T] {
	ctx, cancel := context.WithCancel(context.Background())

	t := &Topic[T]{
		ctx:    ctx,
		cancel: cancel,

		history:           strategy,
		incoming:          make(chan T),
		subscribeEvents:   make(chan *subscription[T]),
		unsubscribeEvents: make(chan *Receiver[T]),
	}

	go t.start()

	return t
}

func (t *Topic[T]) start() {
	for {
		select {
		case <-t.ctx.Done():
			t.mu.Lock()

			for i := range t.subscriptions {
				r := t.subscriptions[i]
				r.notify <- struct{}{}
			}

			t.mu.Unlock()
			return
		case sub := <-t.subscribeEvents:
			t.mu.Lock()
			rx := sub.rx

			if sub.backfill {
				rx.mu.Lock()

				rx.queue = append(rx.queue, t.history.Data()...)
				rx.recv <- struct{}{}

				rx.mu.Unlock()
			}

			t.subscriptions = append(t.subscriptions, rx)

			t.mu.Unlock()
		case rx := <-t.unsubscribeEvents:
			t.mu.Lock()

			for i := range t.subscriptions {
				if t.subscriptions[i] == rx {
					t.subscriptions = append(t.subscriptions[:i], t.subscriptions[i+1:]...)
					break
				}
			}

			t.mu.Unlock()
		case msg := <-t.incoming:
			t.history.Append(msg)

			wait := make(chan struct{})

			go func(el T) {
				wg := sync.WaitGroup{}

				for i := range t.subscriptions {
					wg.Add(1)

					go func(i int) {
						r := t.subscriptions[i]
						r.mu.Lock()

						r.queue = append(r.queue, el)
						if len(r.recv) == 0 {
							r.recv <- struct{}{}
						}

						r.mu.Unlock()
						wg.Done()
					}(i)
				}

				wg.Wait()

				close(wait)
			}(msg)

			<-wait
		}
	}
}

func (t *Topic[T]) Close() {
	t.mu.Lock()
	t.cancel()
	t.mu.Unlock()
}

func (t *Topic[T]) Subscribe(backfill bool) (*Receiver[T], error) {
	select {
	case <-t.ctx.Done():
		return nil, ErrTopicClosed
	default:
		rx := &Receiver[T]{
			topic:  t,
			recv:   make(chan struct{}, 1),
			notify: make(chan struct{}),
		}

		t.subscribeEvents <- newSubscription(rx, backfill)

		return rx, nil
	}
}

func (t *Topic[T]) Unsubscribe(r *Receiver[T]) {
	t.unsubscribeEvents <- r
}

func (t *Topic[T]) Sender() chan<- T {
	return t.incoming
}

func (t *Topic[T]) Send(msg T) {
	t.incoming <- msg
}

func (rx *Receiver[T]) Notify() <-chan struct{} {
	return rx.notify
}

func (rx *Receiver[T]) Recv() <-chan struct{} {
	return rx.recv
}

func (rx *Receiver[T]) Close() {
	rx.mu.Lock()
	rx.topic.Unsubscribe(rx)
	rx.mu.Unlock()
}

func (rx *Receiver[T]) Dequeue() []T {
	rx.mu.Lock()

	if len(rx.queue) == 0 {
		return nil
	}

	clone := rx.queue

	rx.queue = nil
	rx.mu.Unlock()

	return clone
}

func newSubscription[T any](rx *Receiver[T], backfill bool) *subscription[T] {
	return &subscription[T]{
		rx:       rx,
		backfill: backfill,
	}
}
