package eventloop

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/FJSDS/common/logger"
)

func TestNewEventLoop(t *testing.T) {
	r := require.New(t)
	log, err := logger.NewLogger("TestNewEventLoop", ".", zap.DebugLevel)
	r.NoError(err)
	defer log.Flush()
	loop := NewEventLoop(log)
	count := int64(0)
	loop.Start(func(event interface{}) {
		switch e := event.(type) {
		case int64:
			atomic.AddInt64(&count, e)
		}
	})
	loop.PostFunc(func() {
		atomic.AddInt64(&count, 1)
	})
	loop.PostEvent(int64(1))

	loop.Stop()
	r.Equal(int64(2), count)
}

func TestNewEventLoop1(t *testing.T) {
	r := require.New(t)
	log, err := logger.NewLogger("TestNewEventLoop", ".", zap.DebugLevel)
	r.NoError(err)
	defer log.Flush()
	loop := NewEventLoop(log)
	count := int64(0)
	loop.Start(func(event interface{}) {
		switch e := event.(type) {
		case int64:
			atomic.AddInt64(&count, e)
		}
	})
	for i := 0; i < 10; i++ {
		go func() {
			for {
				loop.AfterFuncQueue(time.Microsecond, func() {
					atomic.AddInt64(&count, 1)
				})
			}
		}()
	}
	for i := 0; i < 20; i++ {
		time.Sleep(time.Second)
		fmt.Println(atomic.LoadInt64(&count))
	}
	loop.Stop()

}
