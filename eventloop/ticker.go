package eventloop

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MauriceGit/skiplist"
	"github.com/panjf2000/ants"
	"go.uber.org/zap"

	"github.com/FJSDS/common/logger"
	"github.com/FJSDS/common/timer"
	"github.com/FJSDS/common/utils"
)

type AntsLogger struct {
	log *logger.Logger
}

func (this_ *AntsLogger) Printf(format string, args ...interface{}) {
	this_.log.InfoFormat(format, args...)
}

type Ticker struct {
	m         *skiplist.SkipList
	c         <-chan time.Time
	index     int64
	keyPool   *sync.Pool
	taskPool  *sync.Pool
	lock      *utils.SpinLock
	taskIndex uint32
	log       *logger.Logger
	pool      *ants.Pool
	queue     *EsQueue
}

type taskNode struct {
	fb            func() bool
	f             func()
	t             int64
	tickMill      time.Duration
	isPoolOrQueue int // 0 定时任务线程执行，1 线程池执行，2 queue执行
	Index         float64
}

func (this_ *taskNode) reset() {
	this_.f = nil
	this_.fb = nil
	this_.t = 0
	this_.tickMill = 0
	this_.isPoolOrQueue = 0
	this_.Index = 0
}

func (this_ *taskNode) ExtractKey() float64 {
	return this_.Index
}
func (this_ *taskNode) String() string {
	return fmt.Sprintf("%e", this_.Index)
}

//type nodeKey struct {
//	NanoSecond int64
//	Index      uint64
//}
//
//func (e *nodeKey) ExtractKey() float64 {
//	return float64()
//}
//func (e *nodeKey) String() string {
//	return fmt.Sprintf("%03d", e)
//}

//func (this_ *nodeKey) reset() {
//	this_.NanoSecond = 0
//	this_.Index = 0
//}

//func keyInt64Comparator(a, b interface{}) int {
//	aAsserted := a.(*nodeKey)
//	bAsserted := b.(*nodeKey)
//
//	if aAsserted.NanoSecond > bAsserted.NanoSecond {
//		return 1
//	}
//	if aAsserted.NanoSecond < bAsserted.NanoSecond {
//		return -1
//	}
//	if aAsserted.Index > bAsserted.Index {
//		return 1
//	}
//	if aAsserted.Index < bAsserted.Index {
//		return -1
//	}
//	return 0
//}

func NewTicker(log *logger.Logger) *Ticker {
	pool, _ := ants.NewPool(10000, ants.WithLogger(&AntsLogger{log: log}))
	queue := NewQueue(10000)
	t := newTicker(log, pool, queue)
	t.runQueue()
	t.run()
	return t
}

func newTicker(log *logger.Logger, pool *ants.Pool, queue *EsQueue) *Ticker {
	c, index := timer.GetTickChan()
	sl := skiplist.NewSeedEps(25, 0.000001)
	t := &Ticker{
		c:     c,
		m:     &sl,
		index: index,
		taskPool: &sync.Pool{New: func() interface{} {
			return &taskNode{}
		}},
		log:   log,
		pool:  pool,
		queue: queue,
		lock:  &utils.SpinLock{},
	}
	return t
}

func (this_ *Ticker) GetPool() *ants.Pool {
	return this_.pool
}

func (this_ *Ticker) GetQueue() *EsQueue {
	return this_.queue
}

//func (this_ *ticker) TickerLen() int64 {
//	return int64(len(this_.c))
//}

func (this_ *Ticker) runQueue() {
	xf := func(event interface{}) {
		defer utils.Recover(func(e interface{}) {
			this_.log.Error("event func panic")
		})
		event.(func())()
	}
	go this_.start(xf)

}

func (this_ *Ticker) start(f func(event interface{})) {
	events := make([]interface{}, 4096)
	for {
		gets, _ := this_.queue.Gets(events)
		if gets > 0 {
			es := events[:gets]
			for _, v := range es {
				f(v)
			}
		} else {
			time.Sleep(time.Microsecond * 10)
		}
	}
}

func (this_ *Ticker) run() {

	go func() {
		const maxBatch = 2560
		arr := make([]*taskNode, 0, maxBatch)
		//arrKey := make([]*nodeKey, 0, maxBatch)
		for v := range this_.c {
			now := v.UnixNano()
			for {
				arr = arr[:0]
				this_.lock.Lock()
				for {
					v := this_.m.GetSmallestNode()
					if v == nil {
						break
					}
					vv := v.GetValue()
					nk := vv.(*taskNode)
					if now >= nk.t {
						arr = append(arr, nk)
						this_.m.Delete(vv)
						if len(arr) == maxBatch {
							break
						}
					} else {
						break
					}
				}
				this_.lock.Unlock()
				for _, v := range arr {
					this_.runOneTask(v)
				}
				if len(arr) < maxBatch {
					break
				}
			}
		}
	}()
}

func (this_ *Ticker) runOneTask(t *taskNode) {
	defer utils.Recover(func(e interface{}) {
		this_.log.Error("runOneTask panic", zap.Any("panic info", e))
	})
	if t.f != nil {
		switch t.isPoolOrQueue {
		case 0:
			t.f()
			this_.taskPool.Put(t)
		case 1:
			err := this_.pool.Submit(func() {
				t.f()
				this_.taskPool.Put(t)
			})
			if err != nil {
				this_.log.Error("pool.Submit error", zap.Error(err))
			}
		case 2:
			f := func() {
				t.f()
				this_.taskPool.Put(t)
			}
			ok, _ := this_.queue.Put(f)
			for !ok {
				runtime.Gosched()
				ok, _ = this_.queue.Put(f)
			}
		}
		return
	}
	if t.fb != nil {
		switch t.isPoolOrQueue {
		case 0:
			if t.fb() && t.tickMill > 0 {
				this_.Tick(t.tickMill, t.fb)
			}
			this_.taskPool.Put(t)
		case 1:
			err := this_.pool.Submit(func() {
				if t.fb() && t.tickMill > 0 {
					this_.TickPool(t.tickMill, t.fb)
				}
				this_.taskPool.Put(t)
			})
			if err != nil {
				this_.log.Error("pool.Submit error", zap.Error(err))
			}
		case 2:
			f := func() {
				if t.fb() && t.tickMill > 0 {
					this_.TickQueue(t.tickMill, t.fb)
				}
				this_.taskPool.Put(t)
			}
			ok, _ := this_.queue.Put(f)
			for !ok {
				runtime.Gosched()
				ok, _ = this_.queue.Put(f)
			}
		}
	}
}

func (this_ *Ticker) AfterFunc(d time.Duration, f func()) {
	t := this_.newAfterFunc(d, f)
	this_.add(t)
}

func (this_ *Ticker) AfterFuncPool(d time.Duration, f func()) {
	t := this_.newAfterFunc(d, f)
	t.isPoolOrQueue = 1
	this_.add(t)
}

func (this_ *Ticker) AfterFuncQueue(d time.Duration, f func()) {
	t := this_.newAfterFunc(d, f)
	t.isPoolOrQueue = 2
	this_.add(t)
}

func DurationToMill(d time.Duration) int64 {
	return timer.Now().Add(d).UnixNano() / 1000000
}

func (this_ *Ticker) newAfterFunc(d time.Duration, f func()) *taskNode {
	if d < time.Millisecond {
		d = time.Millisecond
	}

	t := this_.taskPool.Get().(*taskNode)
	t.reset()
	t.t = DurationToMill(d)
	t.f = f
	t.Index = float64(t.t) + this_.newIndex()
	return t
}

func (this_ *Ticker) add(v *taskNode) {
	this_.lock.Lock()
	this_.m.Insert(v)
	this_.lock.Unlock()
}

func (this_ *Ticker) Tick(d time.Duration, fb func() bool) {
	this_.add(this_.newTick(d, fb))
}

func (this_ *Ticker) TickPool(d time.Duration, fb func() bool) {
	t := this_.newTick(d, fb)
	t.isPoolOrQueue = 1
	this_.add(t)
}

func (this_ *Ticker) TickQueue(d time.Duration, fb func() bool) {
	t := this_.newTick(d, fb)
	t.isPoolOrQueue = 2
	this_.add(t)
}

func (this_ *Ticker) newIndex() float64 {
	return float64(atomic.AddUint32(&this_.taskIndex, 1)%1000000) / 1000000
}

func (this_ *Ticker) newTick(d time.Duration, fb func() bool) *taskNode {
	if d < time.Millisecond {
		d = time.Millisecond
	}
	t := this_.taskPool.Get().(*taskNode)
	t.reset()
	t.t = DurationToMill(d)
	t.tickMill = d
	t.fb = fb
	t.Index = float64(t.t) + this_.newIndex()
	return t
}

func (this_ *Ticker) Stop() {
	timer.PutTickChan(this_.index)
}
