package eventloop

import (
	"runtime"
	"sync/atomic"
	"time"

	"github.com/panjf2000/ants"

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
	pool, _ := ants.NewPool(10000, ants.WithLogger(&AntsLogger{log: log}), ants.WithExpiryDuration(time.Second*20))
	t := newTicker(log, pool, queue)
	t.run()
	return &EventLoop{
		queue:    queue,
		timing:   t,
		log:      log,
		workPool: pool,
	}
}

func (this_ *EventLoop) PostEvent(e interface{}) {
	ok, _ := this_.queue.Put(e)
	for !ok {
		runtime.Gosched()
		ok, _ = this_.queue.Put(e)
	}
}

func (this_ *EventLoop) PostFunc(f func()) {
	this_.PostEvent(f)
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

func (this_ *EventLoop) Start(f func(event interface{})) {
	xf := func(event interface{}) {
		defer utils.Recover(func(e interface{}) {
			this_.log.Error("event func panic")
		})
		switch fx := event.(type) {
		case func():
			fx()
		default:
			f(event)
		}
	}
	go this_.start(xf)
}

func (this_ *EventLoop) start(f func(event interface{})) {
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