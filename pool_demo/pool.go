package pool_demo

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrInvalidPoolSize   = errors.New("invalid size for pool")
	ErrInvalidPoolExpiry = errors.New("invalid expiry for pool")
	ErrPoolClosed        = errors.New("pool is closed")
)

// 信号类型
type sig struct{}

// func类型
type f func() error

/*
思考：
1. 如何对worker进行管理
2. 如何定时清理过期worker
3. 如何保证并发时的读写不冲突
4. 分析每个结构体，为什么这么设计， 不能一味的用map和slice， 也不能一味的用chan
5. 运行中的goroutine如何通过release信号停止运行
*/
type Pool struct {
	// 容量
	capacity int32
	// 运行中的数量
	running int32
	// 对每个worker设置过期时间
	expiryDuration time.Duration
	// workers
	workers []*Worker
	// 回收信号
	release chan sig

	lock sync.Mutex
	once sync.Once
}

func NewPool(size int) (*Pool, error) {
	return NewTimingPool(size, 10)
}

func NewTimingPool(size, expiry int) (*Pool, error) {
	if size < 0 {
		return nil, ErrInvalidPoolSize
	}
	if expiry < 0 {
		return nil, ErrInvalidPoolExpiry
	}

	p := &Pool{
		capacity:       int32(size),
		expiryDuration: time.Duration(expiry) * time.Second,
		// 信号使用的异步类型的chan, 长度为1的缓冲区
		release: make(chan sig),
	}
	return p, nil
}

func (p *Pool) Submit(task f) error {
	// 先判断pool是否关闭
	if len(p.release) > 0 {
		return ErrPoolClosed
	}
	// 获取worker
	w := p.getWorker()
	// 将task推送给worker
	// worker的task接收到以后, channel不会再阻塞, 进而再worker的goroutine中执行task, 执行完以后, 继续阻塞, 知道被回收
	w.task <- task
	return nil
}

func (p *Pool) getWorker() *Worker {
	var w *Worker
	waiting := false
	p.lock.Lock()
	idleWorkers := p.workers
	n := len(idleWorkers) - 1

	// 当没有空闲worker时, 判断是否running数量达到了容量上限
	if n < 0 {
		waiting = p.Running() >= p.Cap()
	} else {
		// 拿到最后一个worker, 并删除workers列表的最后一个
		w = idleWorkers[n]
		idleWorkers[n] = nil
		p.workers = idleWorkers[:n]
	}
	p.lock.Unlock()

	// 如果需要等待
	if waiting {
		for {
			// 等待空闲worker
			p.lock.Lock()
			idleWorkers = p.workers
			l := len(idleWorkers) - 1
			if l < 0 {
				p.lock.Unlock()
				continue
			}
			// 一直等到有空闲worker
			w = idleWorkers[l]
			idleWorkers[l] = nil
			p.workers = idleWorkers[:l]
			p.lock.Unlock()
			break
		}
	} else if w == nil {
		// 当运行中的数量小于容量, 新创建一个worker, 并启动
		w := &Worker{
			pool: p,
			task: make(chan f, 1),
		}
		w.run()
		p.incRunning()
	}
	return w
}

// 定期清理
func (p *Pool) periodicallyPurge() {
	heartbeat := time.NewTicker(p.expiryDuration)
	// 1. 使用这种方式, 没有使用select case的形式
	for range heartbeat.C {
		currentTime := time.Now()
		// 2. 加锁了
		p.lock.Lock()
		idleWorkers := p.workers
		if len(idleWorkers) == 0 && p.Running() == 0 && len(p.release) > 0 {
			p.lock.Unlock()
			return
		}
		n := 0
		// 3. 同样是去遍历
		for i, w := range idleWorkers {
			// 4. 使用一样的方式去检查
			if currentTime.Sub(w.recycleTime) <= p.expiryDuration {
				break
			}
			// 5. 因为workers是按照LIFO的顺序, 所以前面的worker都是空闲时间长的
			n = i
			// 回收worker
			w.task <- nil
			// 6. 将worker的位置置为nil, gc可以回收
			idleWorkers[i] = nil
			// 7. 不需要手动修改running, 因为worker回收时, 会修改
		}
		n++
		// 8. 一次性修改workers, 避免了append
		if n >= len(idleWorkers) {
			p.workers = idleWorkers[:0]
		} else {
			p.workers = idleWorkers[n:]
		}
		p.lock.Unlock()
	}
}

func (p *Pool) ReSize(size int) {
	if p.Cap() == size {
		return
	}
	// 先修改容量参数
	atomic.StoreInt32(&p.capacity, int32(size))
	diff := p.Running() - size
	// 当运行数量比新容量大时, 回收workers
	if diff > 0 {
		for i := 0; i < diff; i++ {
			// todo 如果这里的worker都在执行任务中, 可能会造成阻塞, 因为目前还是阻塞型
			// 拿到一个worker, 将worker回收
			w := p.getWorker()
			w.task <- nil
		}
	}
}

func (p *Pool) Running() int {
	return int(atomic.LoadInt32(&p.running))
}

func (p *Pool) Cap() int {
	return int(atomic.LoadInt32(&p.capacity))
}

func (p *Pool) putWorker(w *Worker) {
	w.recycleTime = time.Now()
	p.lock.Lock()
	// 将新worker放进pool中的workers队列
	p.workers = append(p.workers, w)
	p.lock.Unlock()
}

func (p *Pool) incRunning() {
	atomic.AddInt32(&p.running, 1)
}

func (p *Pool) decRunning() {
	atomic.AddInt32(&p.running, -1)
}
