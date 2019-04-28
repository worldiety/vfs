package vfs

import (
	"sync"
	"sync/atomic"
)

// Cancelable is a contract to cancel something. It is introduced to provide an interop with the gomobile world which
// does not work with the go Context because it uses channels and other incompatible declarations.
type Cancelable interface {
	// Cancel it now. If it has ever a consequence, is undefined. However if it has been already cancelled, it
	// has no further effects.
	Cancel()

	// IsCancelled checks if this Cancelable has been cancelled already
	IsCancelled() bool

	// Add appends another cancelable child to this cancelable. If this instance is already
	// cancelled, the added child is immediately cancelled. Otherwise it will get called as soon as
	// #Cancel() is invoked.
	Add(child Cancelable)
}

// A DefaultCancelable just implements the Cancelable contract.
type DefaultCancelable struct {
	mutex     sync.Mutex
	children  []Cancelable
	cancelled int32
}

// Cancel executes immediately all registered cancelables or does nothing if it has been already cancelled.
func (c *DefaultCancelable) Cancel() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	// this is actually too much, but the important thing is get the memory barrier thing correct and
	// ensure that IsCancelled is as fast as possible
	if atomic.CompareAndSwapInt32(&c.cancelled, 0, 1) {
		for _, child := range c.children {
			child.Cancel()
		}
		c.children = nil // avoid memory leaks through callbacks, any others are executed without adding
	}
}

func (c *DefaultCancelable) IsCancelled() bool {
	// do not use mutex to avoid performance degradation due to cache flushes in hot path
	return atomic.LoadInt32(&c.cancelled) != 0
}

// Add appends another cancelable or executes it immediately
func (c *DefaultCancelable) Add(child Cancelable) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if atomic.LoadInt32(&c.cancelled) != 0 {
		child.Cancel()
	} else {
		c.children = append(c.children, child)
	}
}
