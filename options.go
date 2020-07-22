package ants_learning

// todo 绝对的新颖, option的类型是func
// 可能这就是使用指针的原因, 可以将opts的指针传进去, 然后在func里面处理opts
type Option func(opts *Options)

// todo 把func当作参数传进去, 然后对一个对象进行操作, 真牛逼
func loadOptions(options ...Option) *Options {
	opts := new(Options)
	for _, option := range options {
		option(opts)
	}
	return opts
}

// 初始化pool时, 提供的操作项
type Options struct {
	// PreAlloc indicates whether to make memory pre-allocation where initializing Pool.
	// 是否在初始化时, 预加载内存
	PreAlloc bool
}
