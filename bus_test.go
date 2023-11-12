package eventbus_test

import (
	eventbus "hehaowen00/event-bus"
	"log"
	"sync"
	"testing"
	"time"
)

func TestEventBus_1(t *testing.T) {
	bus := eventbus.New[int]()
	wg := sync.WaitGroup{}

	rx1, err := bus.Subscribe(false)
	if err != nil {
		t.Fatal(err)
	}

	rx2, err := bus.Subscribe(false)
	if err != nil {
		t.Fatal(err)
	}

	go startWorker(rx1, 1, &wg)
	go startWorker(rx2, 2, &wg)

	bus.Send() <- 42
	bus.Send() <- 1337

	time.Sleep(time.Second)
	bus.Close()

	wg.Wait()

	if _, err := bus.Subscribe(false); err == nil {
		t.Fatal("expected error")
	}
}

func TestEventBus_2(t *testing.T) {
	bus := eventbus.NewWithHistory(eventbus.NewFixedHistory[int](2))
	wg := sync.WaitGroup{}

	bus.Send() <- 42
	bus.Send() <- 1337

	rx1, err := bus.Subscribe(true)
	if err != nil {
		t.Fatal(err)
	}

	rx2, err := bus.Subscribe(true)
	if err != nil {
		t.Fatal(err)
	}

	go startWorker(rx1, 1, &wg)
	go startWorker(rx2, 2, &wg)

	time.Sleep(time.Second)
	bus.Close()

	wg.Wait()

	if _, err := bus.Subscribe(false); err == nil {
		t.Fatal("expected error")
	}
}

func startWorker[T any](rx *eventbus.Receiver[T], id int, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case <-rx.Notify():
			log.Printf("closed (%d)\n", id)
			return
		case msg, ok := <-rx.Recv():
			if !ok {
				break
			}
			log.Printf("received (%d) %v\n", id, msg)
		}
	}
}
