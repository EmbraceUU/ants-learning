package ants_learning

import "runtime"

const (
	// OPENED represents that the pool is opened.
	OPENED = iota

	// CLOSED represents that the pool is closed.
	CLOSED
)

var (
	// todo 在最大运行数量为1时, workerChan为同步类型, 否则可以允许workerChan有一个缓冲区
	workerChanCap = func() int {
		if runtime.GOMAXPROCS(0) == 1 {
			return 0
		}
		return 1
	}()
)
