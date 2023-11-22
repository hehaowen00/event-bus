package eventbus_test

import (
	"testing"

	eventbus "github.com/hehaowen00/event-bus"
)

func TestEventBus_1(t *testing.T) {
	bus := eventbus.NewEventBus[int]()

	bus.Add("test", eventbus.NewTopic[int]())

	topic := bus.Get("test")
	if topic == nil {
		return
	}

	bus.Close()

	_, err := topic.Subscribe(false)
	if err == nil {
		t.FailNow()
	}
}
