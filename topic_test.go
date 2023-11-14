package eventbus_test

import (
	"fmt"
	eventbus "hehaowen00/event-bus"
	"sync"
	"testing"
	"time"
)

func TestEventBus_1(t *testing.T) {
	topic := eventbus.New[int]()
	wg := sync.WaitGroup{}

	rx1, err := topic.Subscribe(false)
	if err != nil {
		t.Fatal(err)
	}

	rx2, err := topic.Subscribe(false)
	if err != nil {
		t.Fatal(err)
	}

	wg.Add(2)
	go startWorker(rx1, 1, &wg)
	go startWorker(rx2, 2, &wg)

	topic.Send(42)
	topic.Send(1337)

	time.Sleep(time.Second)

	topic.Close()

	wg.Wait()

	if _, err := topic.Subscribe(false); err == nil {
		t.Fatal("expected error")
	}
}

func TestEventBus_2(t *testing.T) {
	hist := eventbus.NewFixedHistory[int](2)
	hist.Prefill([]int{42, 1337})

	topic := eventbus.NewWithHistory(hist)
	wg := sync.WaitGroup{}

	rx1, err := topic.Subscribe(true)
	if err != nil {
		t.Fatal(err)
	}

	rx2, err := topic.Subscribe(true)
	if err != nil {
		t.Fatal(err)
	}

	wg.Add(2)
	go startWorker(rx1, 1, &wg)
	go startWorker(rx2, 2, &wg)

	time.Sleep(time.Second)

	topic.Close()

	wg.Wait()

	if _, err := topic.Subscribe(false); err == nil {
		t.Fatal("expected error")
	}
}

func TestEventBus_3(t *testing.T) {
	topic := eventbus.New[int]()
	wg := sync.WaitGroup{}

	rx, err := topic.Subscribe(false)
	if err != nil {
		t.Fatal(err)
	}

	topic.Send(42)

	wg.Add(1)
	go startWorker(rx, 1, &wg)

	topic.Send(1337)

	time.Sleep(time.Second)

	topic.Close()

	wg.Wait()
}

func TestEventBus_4(t *testing.T) {
	hist := eventbus.NewFixedHistory[int](8)
	hist.Prefill([]int{1, 2, 3, 4, 5, 6, 7, 8})

	topic := eventbus.NewWithHistory(hist)
	wg := sync.WaitGroup{}

	rx1, err := topic.Subscribe(true)
	if err != nil {
		t.Fatal(err)
	}

	rx2, err := topic.Subscribe(true)
	if err != nil {
		t.Fatal(err)
	}

	wg.Add(2)
	go startWorker(rx1, 1, &wg)
	go startWorker(rx2, 2, &wg)

	topic.Send(42)
	topic.Send(1337)

	time.Sleep(time.Second)

	topic.Close()

	wg.Wait()

	if _, err := topic.Subscribe(false); err == nil {
		t.Fatal("expected error")
	}
}

func startWorker[T any](rx *eventbus.Receiver[T], id int, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-rx.Notify():
			return
		case _, ok := <-rx.Recv():
			if !ok {
				continue
			}
			data := rx.Dequeue()
			fmt.Printf("received (%d) %v\n", id, data)
		}
	}
}
