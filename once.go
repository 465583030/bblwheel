package bblwheel

import "sync/atomic"

//Once 实现只执行一次任务的功能，相对原生实现性能更好，
//因为基于CompareAndSwapInt32无锁算法实现
type Once struct {
	done int32
}

//Do ....
func (o Once) Do(f func()) {
	if atomic.LoadInt32(&o.done) == 1 {
		return
	}
	if atomic.CompareAndSwapInt32(&o.done, 0, 1) {
		f()
	}
}
