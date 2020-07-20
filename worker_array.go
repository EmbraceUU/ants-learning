package ants_learning

// todo 将worker列表抽象成一个interface, 根据需求实现不同的workerArray

type workerArray interface {
	len() int
	isEmpty() bool
}

type arrayType int

const (
	stackType arrayType = 1 << iota
	loopQueueType
)
