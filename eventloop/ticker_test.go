package eventloop

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewTicker(t *testing.T) {
	tick := NewTicker(nil)
	defer tick.Stop()
	count := int64(0)
	go func() {
		for i := 0; i < 100000; i++ {
			tick.Tick(time.Millisecond*2, func() bool {
				atomic.AddInt64(&count, 1)
				return true
			})
		}
	}()
	for {
		time.Sleep(time.Second)
		fmt.Println(atomic.LoadInt64(&count))
	}
}
