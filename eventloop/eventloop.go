package eventloop

import (
	"runtime"
	"sync/atomic"
	"time"

	"github.com/panjf2000/ants"
	"go.uber.org/zap"

	"github.com/FJSDS/common/logger"
	"github.com/FJSDS/common/timer"
	"github.com/FJSDS/common/utils"
)

type EventLoop struct {
	queue    *EsQueue
	timing   *Ticker
	stopped  int32
	over     chan struct{}
	log      *logger.Logger
	workPool *ants.Pool
}

func NewEventLoop(log *logger.Logger) *EventLoop {
	queue := NewQueue(100000)
	pool, _ := ants.NewPool(10000, ants.WithPanicHandler(func(i interface{}) {
		log.Error("ants.Pool panic",zap.Any("panic info" ,i))
	}), ants.WithExpiryDuration(time.Second*20))
	t := newTicker(log, pool, queue)
	t.run()
	return &EventLoop{
		queue:    queue,
		timing:   t,
		log:      log,
		workPool: pool,
	}
}

func (this_ *EventLoop) PostEventQueue(e interface{}) {
	ok, _ := this_.queue.Put(e)
	for !ok {
		runtime.Gosched()
		ok, _ = this_.queue.Put(e)
	}
}

func (this_ *EventLoop) PostFuncQueue(f func()) {
	this_.PostEventQueue(f)
}

func (this_ *EventLoop) AfterFuncQueue(d time.Duration, f func()) {
	this_.timing.AfterFuncQueue(d, f)
}

func (this_ *EventLoop) AfterFuncPool(d time.Duration, f func()) {
	this_.timing.AfterFuncPool(d, f)
}

func (this_ *EventLoop) UntilFuncQueue(t time.Time, f func()) {
	this_.timing.AfterFuncQueue(t.Sub(timer.Now()), f)
}

func (this_ *EventLoop) UntilFuncPool(t time.Time, f func()) {
	this_.timing.AfterFuncPool(t.Sub(timer.Now()), f)
}

func (this_ *EventLoop) TickQueue(d time.Duration, f func() bool) {
	this_.timing.TickQueue(d, f)
}

func (this_ *EventLoop) TickPool(d time.Duration, f func() bool) {
	this_.timing.TickPool(d, f)
}

func (this_ *EventLoop) Start(f func(event interface{}), endF ...func()) {
	xf := func(event interface{}) {
		defer utils.Recover(func(e interface{}) {
			this_.log.Error("event func panic", zap.Any("panic info", e))
		})
		switch fx := event.(type) {
		case func():
			fx()
		default:
			f(event)
		}
	}
	var rF []func()
	if len(endF) > 0 {
		for _, v := range endF {
			rF = append(rF, func() {
				defer utils.Recover(func(e interface{}) {
					this_.log.Error("end func panic", zap.Any("panic info", e))
				})
				v()
			})
		}
	}
	go this_.start(xf, rF...)
}

func (this_ *EventLoop) start(f func(event interface{}), endF ...func()) {
	events := make([]interface{}, 4096)
	for atomic.LoadInt32(&this_.stopped) == 0 {
		gets, _ := this_.queue.Gets(events)
		if gets > 0 {
			es := events[:gets]
			for _, v := range es {
				f(v)
			}
		} else {
			time.Sleep(time.Microsecond * 100)
		}
	}
	for {
		v, _, _ := this_.queue.Get()
		if v == nil {
			break
		} else {
			f(v)
		}
	}
	for _, v := range endF {
		v()
	}
	close(this_.over)
}

func (this_ *EventLoop) Stop() {
	this_.timing.Stop()
	this_.workPool.Release()
	this_.over = make(chan struct{})
	atomic.StoreInt32(&this_.stopped, 1)
	<-this_.over
}

func (this_ *EventLoop) Stopped() bool {
	return atomic.LoadInt32(&this_.stopped) == 1
}
