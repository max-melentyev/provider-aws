package instance

import "sync"

// sync.WaitGroup with a helper method.
type WaitGroup struct {
	sync.WaitGroup
}

func (wg *WaitGroup) Spawn(fn func()) {
	wg.Add(1)
	go func() {
		fn()
		wg.Done()
	}()
}
