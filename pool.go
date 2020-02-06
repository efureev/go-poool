package poool

import "sync"

type Pool interface {

	// Queue queues the work to be run, and starts processing immediately
	Queue(fn JobFn) Job

	// Reset reinitializes a pool that has been closed/cancelled back to a working
	// state. if the pool has not been closed/cancelled, nothing happens as the pool
	// is still in a valid running state
	Reset()

	// Cancel cancels any pending work still not committed to processing.
	// Call Reset() to reinitialize the pool for use.
	Cancel()

	// Close cleans up pool data and cancels any pending work still not committed
	// to processing. Call Reset() to reinitialize the pool for use.
	Close()
}

type pool struct {
	m sync.RWMutex

	workers uint
	jobs    chan *job

	cancel chan struct{}
	closed bool
}

// New returns a new pool instance
func New(workers uint) Pool {

	if workers == 0 {
		panic("invalid workers '0'")
	}

	p := &pool{
		workers: workers,
	}

	p.initialize()

	return p
}

func (p *pool) initialize() {

	p.jobs = make(chan *job, p.workers*2)
	p.cancel = make(chan struct{})
	p.closed = false

	// run workers
	for i := 0; i < int(p.workers); i++ {
		go p.newWorker(p.jobs, p.cancel)
	}
}

// Cancel cleans up the pool workers and channels and cancels and pending
// work still yet to be processed.
// call Reset() to reinitialize the pool for use.
func (p *pool) Cancel() {

	err := &ErrCancelled{s: errCancelled}
	p.closeWithError(err)
}

// Close cleans up the pool workers and channels and cancels any pending
// work still yet to be processed.
// call Reset() to reinitialize the pool for use.
func (p *pool) Close() {

	err := &ErrPoolClosed{s: errClosed}
	p.closeWithError(err)
}

func (p *pool) closeWithError(err error) {
	p.m.Lock()

	if !p.closed {
		close(p.cancel)
		close(p.jobs)
		p.closed = true
	}

	for j := range p.jobs {
		j.cancelWithError(err)
	}

	p.m.Unlock()
}

// Queue queues the job to be run, and starts processing immediately
func (p *pool) Queue(fn JobFn) Job {

	j := newJob(fn)

	go func() {
		p.m.RLock()
		defer p.m.RUnlock()

		if p.closed {
			j.err = &ErrPoolClosed{s: errClosed}
			if j.cancelled.Load() == nil {
				close(j.done)
			}

			return
		}

		p.jobs <- j
	}()

	return j
}

// Reset reinitializes a pool that has been closed/cancelled back to a working state.
// if the pool has not been closed/cancelled, nothing happens as the pool is still in
// a valid running state
func (p *pool) Reset() {

	p.m.Lock()
	defer p.m.Unlock()

	if !p.closed {
		return
	}

	// cancelled the pool, not closed it, pool will be usable after calling initialize().
	p.initialize()
}
