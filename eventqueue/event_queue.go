package eventqueue

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"github.com/FJSDS/common/logger"
	"github.com/FJSDS/common/timer"
	"github.com/FJSDS/common/utils"
)

type EventStopped struct{}

type timerFuncInfo struct {
	F func(t time.Time)
	T time.Time
}

type EventQueue struct {
	overQueue     uint32
	overTimeQueue uint32
	stopped       uint32

	queue          chan interface{}
	postQueue      chan interface{}
	wg             *sync.WaitGroup
	timerPostQueue chan *timerFuncInfo
	timerQueue     chan *timerFuncInfo
	log            *logger.Logger

	onceMap map[string]func()
}

func NewEventQueue(cap int, log *logger.Logger) *EventQueue {
	if cap < 10000 {
		cap = 10000
	}

	e := &EventQueue{}
	e.queue = make(chan interface{}, cap)
	e.postQueue = e.queue
	e.timerQueue = make(chan *timerFuncInfo, cap)
	e.timerPostQueue = e.timerQueue
	e.wg = &sync.WaitGroup{}
	e.log = log

	return e
}

func (this_ *EventQueue) setStopped() {
	atomic.StoreUint32(&this_.stopped, 1)
}

func (this_ *EventQueue) Stopped() bool {
	return atomic.LoadUint32(&this_.stopped) == 1
}

func (this_ *EventQueue) Post(msg interface{}) {
	if !this_.Stopped() {
		select {
		case this_.postQueue <- msg:
		default:
		}
	}
}

func (this_ *EventQueue) PostWait(f func()) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	var err error
	this_.Post(func() {
		defer func() {
			if e := recover(); e != nil {
				err = fmt.Errorf("func panic:%+v", e)
				cancel()
			}
		}()
		t, _ := ctx.Deadline()
		if t.Sub(timer.Now()) < time.Millisecond*100 {
			return
		}
		defer cancel()
		select {
		case <-ctx.Done():
		default:
			f()
		}
	})

	<-ctx.Done()
	if err == nil {
		err = ctx.Err()
		if err == context.Canceled {
			err = nil
		}
	}
	return err
}

// ticker 每隔几秒调用,如果函数返回false,则停止
func (this_ *EventQueue) Tick(d time.Duration, f func(time.Time) bool) {
	if !this_.Stopped() {
		time.AfterFunc(d, func() {
			this_.timerPostQueue <- &timerFuncInfo{
				F: func(t time.Time) {
					if f(t) {
						this_.Tick(d, f)
					}
				},
				T: timer.Now(),
			}
		})
	}
}
func (this_ *EventQueue) Once(key string, f func()) {
	this_.Post(func() {
		if this_.onceMap == nil {
			this_.onceMap = map[string]func(){}
		}
		if _, ok := this_.onceMap[key]; !ok {
			this_.onceMap[key] = f
			f()
		}
	})
}

// 多少时间之后调用
func (this_ *EventQueue) AfterFunc(d time.Duration, f func(time.Time)) {
	if !this_.Stopped() {
		if d <= 0 {
			d = time.Nanosecond
		}

		time.AfterFunc(d, func() {
			this_.timerPostQueue <- &timerFuncInfo{
				F: f,
				T: timer.Now(),
			}
		})
	}
}

// 达到某个时间点调用
func (this_ *EventQueue) UntilFunc(t time.Time, f func(time.Time)) {
	if !this_.Stopped() {
		timestamp := t.Sub(timer.Now())
		if timestamp <= 0 {
			timestamp = time.Nanosecond
		}

		time.AfterFunc(timestamp, func() {
			this_.timerPostQueue <- &timerFuncInfo{
				F: f,
				T: timer.Now(),
			}
		})
	}
}

// 达到某个时间点调用
func (this_ *EventQueue) UntilFuncMillSeconds(untilMillSecondTime int64, f func(time.Time)) {
	if !this_.Stopped() {
		timestamp := untilMillSecondTime - timer.NowUnixMillisecond()
		if timestamp <= 0 {
			time.AfterFunc(time.Nanosecond, func() {
				this_.timerPostQueue <- &timerFuncInfo{
					F: f,
					T: timer.Now(),
				}
			})

			return
		}

		time.AfterFunc(time.Duration(timestamp)*time.Millisecond, func() {
			this_.timerPostQueue <- &timerFuncInfo{
				F: f,
				T: timer.Now(),
			}
		})
	}
}

func (this_ *EventQueue) Run(panicF func(e interface{}), f func(event interface{}), endFuncS ...func()) {
	if panicF == nil {
		panicF = func(e interface{}) {
			this_.log.Error("func panic", zap.Any("panic info", e))
		}
	}

	this_.wg.Add(1)
	utils.SafeGO(panicF, func() {
		defer this_.wg.Done()
		defer func() {
			for _, v := range endFuncS {
				v()
			}
		}()

		for {
			select {
			case event, ok := <-this_.queue:
				if ok {
					//this_.log.Info("START event", zap.Any("data", event))
					dealEvent(panicF, event, f)
					//this_.log.Info("END event", zap.Any("data", event))
				} else {
					this_.overQueue++
					this_.queue = nil
					if this_.dealRunGoRoutineStop() {
						return
					}
				}
			case timerF, ok := <-this_.timerQueue:
				if ok {
					utils.RecoverWithFunc(panicF, func() {
						//this_.log.Info("START timerF", zap.String("data", reflect.TypeOf(timerF.F).String()))
						timerF.F(timerF.T)
						//this_.log.Info("END timerF", zap.Any("data", reflect.TypeOf(timerF.F).String()))
					})
				} else {
					this_.overTimeQueue++
					this_.timerQueue = nil
					if this_.dealRunGoRoutineStop() {
						return
					}
				}
			}
		}
	})
}

func (this_ *EventQueue) dealRunGoRoutineStop() (isReturn bool) {
	if this_.overQueue > 0 && this_.overTimeQueue > 0 {
		this_.log.Debug("event queue stopped")
		return true
	}

	return false
}

func dealEvent(panicF func(e interface{}), event interface{}, f func(event interface{})) {
	defer utils.Recover(panicF)

	if event != nil {
		switch e := event.(type) {
		case func():
			if e != nil {
				e()
			}
		default:
			f(event)
		}
	}
}

func (this_ *EventQueue) Stop() {
	this_.Post(&EventStopped{})
	this_.setStopped()
	this_.postQueue = nil
	this_.timerPostQueue = nil
	close(this_.queue)
	close(this_.timerQueue)
	this_.wg.Wait()
}
