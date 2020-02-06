![Go package](https://github.com/efureev/go-poool/workflows/Go%20package/badge.svg)
[![Build Status](https://travis-ci.com/efureev/go-poool.svg?branch=master)](https://travis-ci.com/efureev/go-poool)
[![Go Report Card](https://goreportcard.com/badge/github.com/efureev/go-poool)](https://goreportcard.com/report/github.com/efureev/go-poool)
[![codecov](https://codecov.io/gh/efureev/go-poool/branch/master/graph/badge.svg)](https://codecov.io/gh/efureev/go-poool)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/3db20f8c926442c99a5fbec9227b2176)](https://www.codacy.com/manual/efureev/go-poool?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=efureev/go-poool&amp;utm_campaign=Badge_Grade)
[![Maintainability](https://api.codeclimate.com/v1/badges/491ea6cff45821accb9b/maintainability)](https://codeclimate.com/github/efureev/go-poool/maintainability)

# Workers pool

## Install
```bash
go get -u github.com/efureev/go-poool
```

## Examples

```go
package main

import (
	"time"

	"github.com/efureev/go-poool"
	"github.com/efureev/go-shutdown"
)

func main() {

	p := poool.New(3)

	ticker := time.NewTicker(3 * time.Second)
	fn := createJobFn(100 * time.Millisecond)

	count := 0
	doneJobs := make(chan poool.Job)
	defer close(doneJobs)

	go func(fn func(poool.Job) (interface{}, error)) {
		defer println(`timer stop..`)
		for {
			select {
			case <-ticker.C:
				count++

				for i := 0; i < 5; i++ {
					j := p.Queue(fn)
					println(`>> add job: `, j)
					doneJobs <- j
				}

				if count == 5 {
					ticker.Stop()
					return
				}
			}
		}
	}(fn)

	go func() {
		for j := range doneJobs {
			j.Wait()

			println(`<< job done: `, j.Value(), `, error: `, j.Error())
		}
	}()

	shutdown.
		OnDestroy(func(done chan<- bool) {
			p.Close()
			done <- true
		}).
		Wait()

}

func createJobFn(d time.Duration) poool.JobFn {
	return func(poool.Job) (interface{}, error) {
		time.Sleep(d)
		return true, nil
	}
}
```
