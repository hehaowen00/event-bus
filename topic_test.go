package eventbus_test

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	eventbus "github.com/hehaowen00/event-bus"
)

func TestTopic_1(t *testing.T) {
	topic := eventbus.NewTopic[int]()
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

func TestTopic_2(t *testing.T) {
	hist := eventbus.NewFixedHistory[int](2)
	hist.Append(42)
	hist.Append(1337)

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

func TestTopic_3(t *testing.T) {
	topic := eventbus.NewTopic[int]()
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
	runtime.Gosched()

	topic.Close()

	wg.Wait()
}

func TestTopic_4(t *testing.T) {
	hist := eventbus.NewFixedHistory[int](8)
	for i := 1; i < 9; i++ {
		hist.Append(i)
	}

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

type TestMessage struct {
	data string
}

func TestTopic_5(t *testing.T) {
	hist := eventbus.NewFixedHistory[TestMessage](8)
	hist.Append(TestMessage{
		data: "Hello, World!",
	})

	topic := eventbus.NewWithHistory(hist)
	wg := sync.WaitGroup{}

	rx1, err := topic.Subscribe(false)
	if err != nil {
		t.Fatal(err)
	}

	topic.Send(TestMessage{
		data: "Hello, World!",
	})

	wg.Add(1)
	go startWorker[TestMessage](rx1, 1, &wg)

	topic.Send(TestMessage{
		data: "Hello, World!",
	})

	topic.Send(TestMessage{
		data: "Hello, World!",
	})

	time.Sleep(time.Second)

	topic.Close()
	wg.Wait()
}

func startWorker[T any](rx *eventbus.Receiver[T], id int, wg *sync.WaitGroup) {
	for {
		select {
		case <-rx.Notify():
			wg.Done()
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
