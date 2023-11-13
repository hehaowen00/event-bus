package eventbus

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestHistory_1(t *testing.T) {
	hist := NewFixedHistory[int](2)
	hist.Prefill([]int{42, 1337})
	hist.append(0)

	ch := make(chan int)
	hist.fill(ch)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		data := []int{}
		for {
			select {
			case <-ctx.Done():
				return
			case v := <-ch:
				data = append(data, v)
				fmt.Println("received", v)
			}
		}
	}()

	time.Sleep(time.Second)
	cancel()
}
