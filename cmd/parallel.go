package main

import "sync"

type parallel struct {
	fns []func()
}

func (p *parallel) add(fn func()) {
	p.fns = append(p.fns, fn)
}

func (p *parallel) wait() {
	wg := &sync.WaitGroup{}
	wg.Add(len(p.fns))
	for _, fn := range p.fns {
		go func(fn func()) {
			defer wg.Done()
			fn()
		}(fn)
	}

	wg.Wait()
}

func newParallel() parallel {
	return parallel{fns: make([]func(), 0)}
}
