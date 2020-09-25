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
		for i := 0; i < 100000000; i++ {
			tick.AfterFuncQueue(time.Millisecond*2, func() {
				atomic.AddInt64(&count, 1)
			})
		}
	}()

	old := int64(0)
	for {
		time.Sleep(time.Second)
		n := atomic.LoadInt64(&count)
		fmt.Println(n, n-old)
		old = n
	}
}
