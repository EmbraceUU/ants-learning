package ants_learning

import (
	"sync"
	"time"
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
	// todo 增加了一个缓存, 可以加速获取可用worker
	workerCache sync.Pool
	// blockingNum is the number of the goroutines already been blocked
	// todo 添加了一个阻塞的数量
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
	// 创建一个goWorker
	// 查看p是否release了
	if p.state != 0 {
		return nil
	}
	// 检查workers的数量, 如果有,返回worker,
	if !p.workers.isEmpty() {
		return p.workers.detach()
	}
	// 如果没有, 检查running的数量, 是否超过了容量
	// 如果超过了容量, 等待空闲worker
	if p.running >= p.capacity {
		for {
			if !p.workers.isEmpty() {
				return p.workers.detach()
			}
		}
	}
	// 如果没有超过容量, 创建一个goWorker, 然后返回, 并增加running
	w = &goWorker{
		pool:        p,
		task:        make(chan func(), 1),
		recycleTime: time.Now(),
	}
	w.run()
	return nil
}

// revertWorker puts a worker back into free pool, recycling the goroutines.
func (p *Pool) revertWorker(worker *goWorker) bool {
	return false
}
