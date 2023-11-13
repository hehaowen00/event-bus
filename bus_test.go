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
	hist := eventbus.NewFixedHistory[int](2)
	hist.Prefill([]int{42, 1337})

	bus := eventbus.NewWithHistory(hist)
	wg := sync.WaitGroup{}

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

func TestEventBus_3(t *testing.T) {
	bus := eventbus.New[int]()
	wg := sync.WaitGroup{}
	wg2 := sync.WaitGroup{}

	rx, err := bus.Subscribe(false)
	if err != nil {
		t.Fatal(err)
	}

	wg.Add(2)

	go func() {
		defer wg.Done()
		bus.Send() <- 42
	}()

	go startWorker(rx, 1, &wg2)

	go func() {
		defer wg.Done()
		bus.Send() <- 1337
	}()

	wg.Wait()

	time.Sleep(time.Second)

	bus.Close()

	wg2.Wait()
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
