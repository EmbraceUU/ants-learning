package ants_learning

import "sync"

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
	return nil, nil
}
