package poool

import (
	"sync"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestPool(t *testing.T) {
	Convey("create Pool", t, func() {
		var res []Job

		pool := New(4)
		defer pool.Close()

		newJobFunc := func(d time.Duration) JobFn {
			return func(Job) (interface{}, error) {
				time.Sleep(d)
				return nil, nil
			}
		}

		Convey("Run Jobs", func() {
			for i := 0; i < 10; i++ {
				j := pool.Queue(newJobFunc(time.Second * 1))
				res = append(res, j)
			}

			var count int

			for _, j := range res {
				j.Wait()

				So(j.Error(), ShouldBeNil)
				So(j.Value(), ShouldBeNil)
				count++
			}

			So(count, ShouldEqual, 10)
		})

		pool.Close() // testing no error occurs as Close will be called twice once defer pool.Close() fires

	})
}

func TestCancel(t *testing.T) {

	Convey("Pool: test Cancel", t, func() {

		m := new(sync.RWMutex)
		var closed bool
		c := make(chan Job, 100)

		pool := New(4)
		defer pool.Close()

		newFunc := func(d time.Duration) JobFn {
			return func(Job) (interface{}, error) {
				time.Sleep(d)
				return 1, nil
			}
		}

		go func(ch chan Job) {
			for i := 0; i < 40; i++ {

				go func(ch chan Job) {
					m.RLock()
					if closed {
						m.RUnlock()
						return
					}

					ch <- pool.Queue(newFunc(time.Second * 1))
					m.RUnlock()
				}(ch)
			}
		}(c)

		time.Sleep(time.Second * 1)
		pool.Cancel()
		m.Lock()
		closed = true
		close(c)
		m.Unlock()

		var count int

		for j := range c {
			j.Wait()

			if j.Error() != nil {
				_, ok := j.Error().(*ErrCancelled)
				if !ok {
					_, ok = j.Error().(*ErrPoolClosed)
					if ok {
						So(j.Error().Error(), ShouldEqual, "ERROR: Job added/run after the pool had been closed or cancelled")
					}
				} else {
					So(j.Error().Error(), ShouldEqual, "ERROR: Job Cancelled")
				}

				So(ok, ShouldBeTrue)

				continue
			}

			count += j.Value().(int)
		}

		So(count, ShouldNotEqual, 40)

		// reset and test again
		pool.Reset()

		wrk := pool.Queue(newFunc(time.Millisecond * 300))
		wrk.Wait()

		_, ok := wrk.Value().(int)
		So(ok, ShouldBeTrue)

		wrk = pool.Queue(newFunc(time.Millisecond * 300))
		time.Sleep(time.Second * 1)
		wrk.Cancel()
		wrk.Wait() // proving we don't get stuck here after cancel
		So(wrk.Error(), ShouldBeNil)

		pool.Reset() // testing that we can do this and nothing bad will happen as it checks if pool closed

		pool.Close()

		wu := pool.Queue(newFunc(time.Second * 1))
		wu.Wait()

		So(wu.Error(), ShouldNotBeNil)
		So(wu.Error(), ShouldBeError)

		So(wu.Error().Error(), ShouldEqual, "ERROR: Job added/run after the pool had been closed or cancelled")
	})
}

func TestBadWorkerCount(t *testing.T) {
	Convey("create Pool without workers", t, func() {
		So(func() { New(0) }, ShouldPanicWith, "invalid workers '0'")
	})
}

func TestPanicRecovery(t *testing.T) {

	Convey("Test Panic recovery", t, func() {
		pool := New(2)
		defer pool.Close()

		newFunc := func(d time.Duration, i int) JobFn {
			return func(Job) (interface{}, error) {
				if i == 1 {
					panic("OMG OMG OMG! something bad happened!")
				}
				time.Sleep(d)
				return 1, nil
			}
		}

		var j Job
		for i := 0; i < 4; i++ {
			time.Sleep(time.Second * 1)
			if i == 1 {
				j = pool.Queue(newFunc(time.Second*1, i))
				continue
			}
			pool.Queue(newFunc(time.Second*1, i))
		}

		j.Wait()

		So(j.Error(), ShouldBeError)
		So(j.Error().Error()[:84], ShouldEqual, "ERROR: Job failed due to a recoverable error: 'OMG OMG OMG! something bad happened!'")
	})
}

func BenchmarkPoolSmallRun(b *testing.B) {

	res := make([]Job, 10)

	b.ReportAllocs()

	pool := New(10)
	defer pool.Close()

	fn := func(j Job) (interface{}, error) {
		time.Sleep(time.Millisecond * 500)
		if j.IsCancelled() {
			return nil, nil
		}
		time.Sleep(time.Millisecond * 500)
		return 1, nil
	}

	for i := 0; i < 10; i++ {
		res[i] = pool.Queue(fn)
	}

	var count int

	for _, cw := range res {

		cw.Wait()

		if cw.Error() == nil {
			count += cw.Value().(int)
		}
	}

	if count != 10 {
		b.Fatal("Count Incorrect")
	}
}

func BenchmarkPoolSmallCancel(b *testing.B) {

	res := make([]Job, 0, 20)

	b.ReportAllocs()

	pool := New(4)
	defer pool.Close()

	newFunc := func(i int) JobFn {
		return func(j Job) (interface{}, error) {
			time.Sleep(time.Millisecond * 500)
			if j.IsCancelled() {
				return nil, nil
			}
			time.Sleep(time.Millisecond * 500)
			return i, nil
		}
	}

	for i := 0; i < 20; i++ {
		if i == 6 {
			pool.Cancel()
		}
		res = append(res, pool.Queue(newFunc(i)))
	}

	for _, wrk := range res {
		if wrk == nil {
			continue
		}
		wrk.Wait()
	}
}

func BenchmarkPoolOverConsumeLargeRun(b *testing.B) {

	res := make([]Job, 100)

	b.ReportAllocs()

	pool := New(25)
	defer pool.Close()

	newFunc := func(i int) JobFn {
		return func(j Job) (interface{}, error) {
			time.Sleep(time.Millisecond * 500)
			if j.IsCancelled() {
				return nil, nil
			}
			time.Sleep(time.Millisecond * 500)
			return 1, nil
		}
	}

	for i := 0; i < 100; i++ {
		res[i] = pool.Queue(newFunc(i))
	}

	var count int

	for _, cw := range res {

		cw.Wait()

		count += cw.Value().(int)
	}

	if count != 100 {
		b.Fatalf("Count Incorrect, Expected '100' Got '%d'", count)
	}
}

func BenchmarkPoolLargeCancel(b *testing.B) {

	res := make([]Job, 0, 1000)

	b.ReportAllocs()

	pool := New(4)
	defer pool.Close()

	newFunc := func(i int) JobFn {
		return func(j Job) (interface{}, error) {
			time.Sleep(time.Millisecond * 500)
			if j.IsCancelled() {
				return nil, nil
			}
			time.Sleep(time.Millisecond * 500)
			return i, nil
		}
	}

	for i := 0; i < 1000; i++ {
		if i == 6 {
			pool.Cancel()
		}
		res = append(res, pool.Queue(newFunc(i)))
	}

	for _, wrk := range res {
		if wrk == nil {
			continue
		}
		wrk.Wait()
	}
}
