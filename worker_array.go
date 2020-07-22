package ants_learning

import "time"

// todo 将worker列表抽象成一个interface, 根据需求实现不同的workerArray

type workerArray interface {
	len() int
	isEmpty() bool
	insert(worker *goWorker) error
	// 剥离, 拆分的意思, 应该是get a worker and delete from workers的意思
	detach() *goWorker
	retrieveExpiry(duration time.Duration) []*goWorker
	reset()
}

type arrayType int

const (
	stackType arrayType = 1 << iota
	loopQueueType
)

// 根据不同需求, 新建不同的workerArray
func newWorkerArray(aType arrayType, size int) workerArray {
	switch aType {
	case stackType:
		return nil
	case loopQueueType:
		return nil
	default:
		return nil
	}
}
