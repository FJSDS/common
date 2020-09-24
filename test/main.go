package main

import (
	"fmt"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/FJSDS/common/eventloop"
	"github.com/FJSDS/common/logger"
	"github.com/FJSDS/common/timer"
)

func main() {
	log, _ := logger.NewLogger("test", ".", zap.DebugLevel)
	loop := eventloop.NewEventLoop(log)
	defer loop.Stop()
	loop.Start(func(event interface{}) {

	})
	count := int64(0)
	go func() {
		for i := 0; i < 50000; i++ {
			loop.TickQueue(time.Millisecond*100, func() bool {
				atomic.AddInt64(&count, 1)
				return true
			})
		}
		//for i := 0; i < 50000; i++ {
		//	loop.TickPool(time.Millisecond*100, func() bool {
		//		atomic.AddInt64(&count, 1)
		//		return true
		//	})
		//}
		for {
			loop.PostFunc(func() {
				atomic.AddInt64(&count, 1)
			})
			loop.PostFunc(func() {
				atomic.AddInt64(&count, 1)
			})
			loop.AfterFuncQueue(time.Millisecond*500, func() {
				atomic.AddInt64(&count, 1)
			})
		}
	}()

	old := int64(0)
	for {
		time.Sleep(time.Second)
		n := atomic.LoadInt64(&count)
		fmt.Println(n, n-old, timer.Now())
		old = n
	}
}
