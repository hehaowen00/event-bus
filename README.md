# event-bus

A golang channel based event bus that tries to guarantee the same message order for all subscribers.

## usage

```go
// create event bus with empty history
bus := eventbus.New[int]()

// subscribe
rx, err := bus.Subscribe(false)
_ = err

// start worker
go func() {
    for {
        select {
            case <- rx.Notify():
                return
            case msg <- rx.Recv():
                fmt.Println(msg)
        }
    }
}()

// send message
bus.Send(42)

bus.Close()
```

## message history

- EmptyHistory - messages are never stored and aren't sent to new subscribers
```go
history := NewEmptyHistory()
bus := eventbus.NewWithHistory(history)
```

- FixedHistory - a fixed amount of messages are stored and sent to new subscribers
```go
history := NewFixedHistory(len)
bus := eventbus.NewWithHistory(history)
```

- UnboundedHistory - all messages are stored and sent to new subscribers
```go
history := NewUnboundedHistory()
bus := eventbus.NewWithHistory(history)
```
