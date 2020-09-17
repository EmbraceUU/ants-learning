package ants_learning

import (
	"sync"
	"sync/atomic"
	"time"
)

type Pool struct {
	capacity int32

	running int32

	workers workerArray

	// state is used to notice the pool to closed itself
	state int32

	// lock for synchronous operation
	// todo 自己实现的锁
	lock sync.Locker

	// cond for waiting a idle worker
	//  todo 这个适用于连续多个的信号或者广播 ?
	cond *sync.Cond

	// workerCache speeds up the obtainment of an usable worker in function: retrieveWorker
	// todo 增加了一个cache, 总结一下sync.Pool的应用和优点
	workerCache sync.Pool

	// blockingNum is the number of the goroutines already been blocked
	// todo 需要总结一下它的阻塞机制, 在检索可用worker时, 如何起作用的
	blockingNum int

	// todo 为什么用指针? 这是一种设计模式, 可以看到micro中也用到了类似的设计, 并且公司项目中也有类似的设计
	// options like config
	options *Options
}

// periodicallyPurge clears expired workers periodically.
func (p *Pool) purgePeriodically() {
	// 思路: 遍历空闲worker, 检查空闲worker的时间戳是否超时, 将超时的释放掉
	// 根据LIFO的规则, 遍历worker一直到发现没有过期的那一个, 将前面的全部释放掉, 这样就不会发生复制了
	heartbeat := time.NewTicker(p.options.ExpiryDuration)
	defer heartbeat.Stop()

	// 定时任务: 定时清理过期worker
	// todo 这种方式不如使用 done-channel 及时, 当pool关闭时, 需要等一个ExpiryDuration才能回收该协程
	for range heartbeat.C {
		if atomic.LoadInt32(&p.state) == CLOSED {
			break
		}

		p.lock.Lock()
		expiredWorkers := p.workers.retrieveExpiry(p.options.ExpiryDuration)
		p.lock.Unlock()

		// Notify obsolete workers to stop.
		// This notification must be outside the p.lock, since w.task
		// may be blocking and may consume a lot of time if many workers
		// are located on non-local CPUs.
		for i := range expiredWorkers {
			expiredWorkers[i].task <- nil
			expiredWorkers[i] = nil
		}

		// There might be a situation that all workers have been cleaned up(no any worker is running)
		// while some invokers still get stuck in "p.cond.Wait()",
		// then it ought to wakes all those invokers.
		if p.Running() == 0 {
			p.cond.Broadcast()
		}
	}
}

// 新创建一个pool
func NewPool(size int, options ...Option) (*Pool, error) {
	opts := loadOptions(options...)
	if size < 0 {
		size = -1
	}
	p := &Pool{
		capacity: int32(size),
		lock:     new(sync.RWMutex),
		options:  opts,
	}
	p.workerCache.New = func() interface{} {
		return &goWorker{
			pool: p,
			task: make(chan func(), workerChanCap),
		}
	}
	// 开启一个协程, 专门清理workers
	go p.purgePeriodically()
	return p, nil
}

// Submit submits a task to this pool.
func (p *Pool) Submit(task func()) error {
	if atomic.LoadInt32(&p.state) == CLOSED {
		return ErrPoolClosed
	}

	var w *goWorker
	if w = p.retrieveWorker(); w == nil {
		return ErrPoolOverload
	}

	w.task <- task
	return nil
}

// Running returns the number of the currently running goroutines.
func (p *Pool) Running() int {
	return int(atomic.LoadInt32(&p.running))
}

// Free returns the available goroutines to work.
func (p *Pool) Free() int {
	return p.Cap() - p.Running()
}

// Cap returns the capacity of this pool.
func (p *Pool) Cap() int {
	return int(atomic.LoadInt32(&p.capacity))
}

// Tune changes the capacity of this pool, this method is noneffective to the infinite pool.
func (p *Pool) Tune(size int) {
	if capacity := p.Cap(); capacity == -1 || size <= 0 || size == capacity || p.options.PreAlloc {
		return
	}
	atomic.StoreInt32(&p.capacity, int32(size))
}

// Release Closes this pool.
func (p *Pool) Release() {
	atomic.StoreInt32(&p.state, CLOSED)
	p.lock.Lock()
	p.workers.reset()
	p.lock.Unlock()
}

// Reboot reboots a released pool.
func (p *Pool) Reboot() {
	// todo 00002
	if atomic.CompareAndSwapInt32(&p.state, CLOSED, OPENED) {
		go p.purgePeriodically()
	}
}

func (p *Pool) incRunning() {

}

func (p *Pool) decRunning() {

}

// retrieveWorker returns a available worker to run the tasks.
func (p *Pool) retrieveWorker() (w *goWorker) {
	spawnWorker := func() {
		w = p.workerCache.Get().(*goWorker)
		w.run()
	}
	p.lock.Lock()

	w = p.workers.detach()

	if w != nil {
		p.lock.Unlock()
	} else if capacity := p.Cap(); capacity == -1 {
		p.lock.Unlock()
		spawnWorker()
	} else if p.Running() < capacity {
		p.lock.Unlock()
		spawnWorker()
	} else {
		if p.options.Nonblocking {
			p.lock.Unlock()
			return
		}
	Reentry:
		// todo 00001
		if p.options.MaxBlockingTasks != 0 && p.options.MaxBlockingTasks >= p.blockingNum {
			p.lock.Unlock()
			return
		}
		p.blockingNum++
		p.cond.Wait()
		p.blockingNum--

		if p.Running() == 0 {
			p.lock.Unlock()
			spawnWorker()
			return
		}

		w = p.workers.detach()
		if w == nil {
			goto Reentry
		}

		p.lock.Unlock()
	}
	return
}

// revertWorker puts a worker back into free pool, recycling the goroutines.
// 将worker放回空闲队列, 复用协程
func (p *Pool) revertWorker(worker *goWorker) bool {
	// 检查
	if capacity := p.Cap(); capacity > p.Running() || atomic.LoadInt32(&p.state) == CLOSED {
		return false
	}
	worker.recycleTime = time.Now()
	p.lock.Lock()
	// 把worker插入workers
	err := p.workers.insert(worker)
	if err != nil {
		p.lock.Unlock()
		return false
	}

	// 因为回收了一个worker, 所以释放一个信号, 允许一个被wait阻塞的invoker被唤醒
	p.cond.Signal()
	p.lock.Unlock()
	return true
}
