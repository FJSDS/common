package eventqueue

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yireyun/go-queue"
	"go.uber.org/zap"

	"github.com/FJSDS/common/logger"
)

func TestNewEventQueue(t *testing.T) {
	r := require.New(t)
	log, err := logger.NewLogger("test_eventqueue", ".", zap.DebugLevel)
	r.NoError(err)
	queue := NewEventQueue(0, log)
	queue.Run(nil, func(event interface{}) {

	})
	start := time.Now()
	for i := 0; i < 1000000; i++ {
		queue.AfterFunc(time.Nanosecond, func(t time.Time) {
		})
	}
	time.Sleep(time.Second)
	queue.Stop()
	fmt.Println(time.Now().Sub(start))
}

func TestLockFreeQueue(t *testing.T) {
	//r := require.New(t)
	q := queue.NewQueue(100000)
	for x := 0; x < 10; x++ {
		go func() {
			for i := 1; i <= 100000000; i++ {
				for {
					ok, _ := q.Put(i)
					if ok {
						break
					} else {
						time.Sleep(time.Microsecond)
					}
				}

			}
		}()
	}

	start := time.Now()
	count := 0
	for {
		v, ok, _ := q.Get()
		if v != nil {
			count++
			if count%100000000 == 0 {
				fmt.Println(count)
			}
		} else {
			if !ok {
				time.Sleep(time.Microsecond)
			}
		}
	}
	fmt.Println(time.Now().Sub(start))
}
