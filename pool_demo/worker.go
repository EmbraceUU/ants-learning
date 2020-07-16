package pool_demo

import "time"

type Worker struct {
	// pool who owns this worker
	// 这里包含了pool的指针，而pool里面也有worker的指针
	pool        *Pool
	task        chan f
	recycleTime time.Time
}

func (w *Worker) run() {
	// 这里是新开了一个goroutine去执行task
	// pool和work本身并没有包含goroutine, 而是去控制goroutine的开启和关闭
	go func() {
		for f := range w.task {
			if f == nil {
				w.pool.decRunning()
				return
			}
			_ = f()
			w.pool.putWorker(w)
		}
	}()
}
