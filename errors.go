package poool

const (
	errCancelled = "ERROR: Job Cancelled"
	errClosed    = "ERROR: Job added/run after the pool had been closed or cancelled"
	errRecovery  = "ERROR: Job failed due to a recoverable error: '%v'\n, Stack Trace:\n %s"
)

// ErrPoolClosed is the error returned to all jobs that may have been in or added to the pool after it's closing.
type ErrPoolClosed struct {
	s string
}

// Error prints Job Close error
func (e *ErrPoolClosed) Error() string {
	return e.s
}

// ErrCancelled is the error returned to a Job when it has been cancelled.
type ErrCancelled struct {
	s string
}

// Error prints Job Cancellation error
func (e *ErrCancelled) Error() string {
	return e.s
}

// ErrRecovery contains the error when a consumer goroutine needed to be recovers
type ErrRecovery struct {
	s string
}

// Error prints recovery error
func (e *ErrRecovery) Error() string {
	return e.s
}
