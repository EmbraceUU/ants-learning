package ants_learning

import (
	"sync"
)

type Pool struct {
	// pool容量
	capacity int32
	// 当前运行中的goroutine数量
	running int32
	// workers is a slice that store the available workers
	// 存储可获得的worker列表
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
	// todo 增加了一个cache, 有时间总结一下sync.Pool的应用和优点
	workerCache sync.Pool
	// blockingNum is the number of the goroutines already been blocked
	// todo 需要总结一下它的阻塞机制, 在检索可用worker时, 如何起作用的
	blockingNum int

	// 操作项
	// todo 为什么用指针?
	options *Options
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
	return p, nil
}

// 提交一个task
func (p *Pool) Submit(task func()) error {
	// 获取一个worker
	// 将task推送给worker
	return nil
}

func (p *Pool) Running() int {
	return 0
}

func (p *Pool) Free() int {
	return 0
}

func (p *Pool) Cap() int {
	return 0
}

// Tune changes the capacity of this pool, this method is noneffective to the infinite pool.
func (p *Pool) Tune(size int) {

}

func (p *Pool) Release() {

}

func (p *Pool) Reboot() {

}

func (p *Pool) incRunning() {

}

func (p *Pool) decRunning() {

}

// retrieveWorker returns a available worker to run the tasks.
// 返回一个可用的worker
func (p *Pool) retrieveWorker() (w *goWorker) {
	// 定义一个func, 用来从cache中获取一个worker, 并运行它
	spawnWorker := func() {
		w = p.workerCache.Get().(*goWorker)
		w.run()
	}
	// todo 忘记了加锁, 这里只能允许同时一个线程或者协程检索worker
	p.lock.Lock()

	// 获取一个worker
	w = p.workers.detach()
	// 如果不为Nil, 说明有空闲worker, 直接return了
	if w != nil {
		p.lock.Unlock()
	} else if capacity := p.Cap(); capacity == -1 {
		p.lock.Unlock()
		spawnWorker()
		// 如果当前运行的worker数量 < p的容量, 可以新建一个worker
	} else if p.Running() < capacity {
		p.lock.Unlock()
		spawnWorker()
	} else {
		if p.options.Nonblocking {
			p.lock.Unlock()
			return
		}
	Reentry:
		// todo 疑问: 如果在retrieveWorker开始就加了锁, 那么应该只有一个协程可以进到这里, 那么blockingNum还有什么意义呢 ?
		// 我认为这个问题非常明显, 应该是我想错了
		if p.options.MaxBlockingTasks != 0 && p.options.MaxBlockingTasks >= p.blockingNum {
			p.lock.Unlock()
			return
		}
		// todo 了解cond
		p.blockingNum++
		p.cond.Wait()
		p.blockingNum--

		// 尝试获取worker
		w = p.workers.detach()
		if w == nil {
			// todo 使用goto代替了for
			goto Reentry
		}
		p.lock.Unlock()
	}
	return
}

// revertWorker puts a worker back into free pool, recycling the goroutines.
func (p *Pool) revertWorker(worker *goWorker) bool {
	return false
}
