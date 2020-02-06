package poool

import (
	"sync/atomic"
)

type Job interface {
	// Wait blocks until Job has been processed or cancelled
	Wait()

	// Value returns the job return value
	Value() interface{}

	// Error returns the Job's error
	Error() error

	// Cancel cancels this specific job, if not already committed
	// to processing.
	Cancel()

	// IsReadyForWork displays Job can execute
	IsReadyForWork() bool

	// IsCancelled returns if the Job has been cancelled.
	// NOTE: After Checking IsCancelled(), if it returns false the
	// Job can no longer be cancelled and will use your returned values.
	IsCancelled() bool
}

// JobFn user function for executing
type JobFn func(j Job) (interface{}, error)

// Job is control construct for Job
type job struct {
	done chan struct{}
	fn   JobFn

	value interface{}
	err   error

	writing    atomic.Value
	cancelling atomic.Value
	cancelled  atomic.Value
}

// Cancel cancels this specific job, if not already committed to processing.
func (j *job) Cancel() {
	j.cancelWithError(&ErrCancelled{s: errCancelled})
}

// cancelWithError cancels this specific job with custom error
func (j *job) cancelWithError(err error) {

	j.cancelling.Store(struct{}{})

	if j.writing.Load() == nil && j.cancelled.Load() == nil {
		j.cancelled.Store(struct{}{})
		j.err = err
		close(j.done)
	}
}

// Wait blocks until Job has been processed or cancelled
func (j *job) Wait() {
	<-j.done
}

// Error returns the Job's error
func (j *job) Error() error {
	return j.err
}

// Value returns the jobs return value
func (j *job) Value() interface{} {
	return j.value
}

// IsReadyForWork displays Job can execute
func (j *job) IsReadyForWork() bool {
	return j.cancelled.Load() == nil && j.cancelling.Load() == nil
}

// IsCancelled returns if the Job has been cancelled.
// NOTE: After Checking IsCancelled(), if it returns false the
// Job can no longer be cancelled and will use your returned values.
func (j *job) IsCancelled() bool {
	j.writing.Store(struct{}{}) // ensure that after this check we are committed as cannot be cancelled if not already
	return j.cancelled.Load() != nil
}

func newJob(fn JobFn) *job {
	return &job{
		done: make(chan struct{}),
		fn:   fn,
	}
}
