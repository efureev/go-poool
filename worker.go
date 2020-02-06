package poool

import (
	"fmt"
	"math"
	"runtime"
)

// passing job and cancel channels to newWorker() to avoid any potential race condition
// between p.work read & write
func (p *pool) newWorker(jobs chan *job, cancel chan struct{}) {

	var j *job

	defer func(p *pool) {
		if err := recover(); err != nil {

			trace := make([]byte, 1<<16)
			n := runtime.Stack(trace, true)

			s := fmt.Sprintf(errRecovery, err, string(trace[:int(math.Min(float64(n), float64(7000)))]))

			iwu := j
			iwu.err = &ErrRecovery{s: s}
			close(iwu.done)

			// need to fire up new worker to replace this one as this one is exiting
			p.newWorker(p.jobs, p.cancel)
		}
	}(p)

	var value interface{}
	var err error

	for {
		select {
		case j = <-jobs:

			// possible for one more nilled out value to make it
			// through when channel closed, don't quite understand the why
			if j == nil {
				continue
			}

			// support for individual Job cancellation
			// and batch job cancellation
			if j.cancelled.Load() == nil {
				value, err = j.fn(j)

				j.writing.Store(struct{}{})

				// need to check again in case the Job cancelled this job
				// otherwise we'll have a race condition
				if j.IsReadyForWork() {
					j.value, j.err = value, err

					// who knows where the Done channel is being listened to on the other end
					// don't want this to block just because caller is waiting on another unit
					// of work to be done first so we use close
					close(j.done)
				}
			}

		case <-cancel:
			return
		}
	}
}
