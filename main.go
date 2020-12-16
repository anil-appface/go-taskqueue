package main

import (
	"bytes"
	"container/list"
	"fmt"
	"sync"
	"time"
)

// Worker allows running jobs with a queue.
type Worker struct {
	lock    *sync.Mutex
	queue   *list.List
	maxJobs int

	cleanup chan interface{}
	stop    chan struct{}
	done    chan struct{}
}

// NewWorker creates a new worker
func NewWorker(maxJobs int) *Worker {
	fmt.Println("creating worker object")
	return &Worker{queue: list.New(),
		maxJobs: maxJobs,
		lock:    &sync.Mutex{},
		cleanup: make(chan interface{}),
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}

}

// Execute the worker jobs
func (w *Worker) Execute() {
	defer func() {
		close(w.done)
	}()
	c := &Context{}

	// Release the buffer every 15 minutes
	release := time.NewTicker(time.Minute * 15)

	for {
		select {
		case <-w.stop:
			// Cleanup and return
			release.Stop()
			return
		case <-release.C:
			// Release the buffer
			c.Buffer = nil
		default:
			task := w.pop()
			if task == nil {
				// <-w.notifier
				continue
			}

			//Run
			task.RunTask()
			w.cleanup <- task
		}
	}
}

// Cleanup the worker jobs
func (w *Worker) Cleanup() {

	for {
		select {
		case <-w.done:
			fmt.Println("done in cleanup")
			//cleanup done.
			return
		case task := <-w.cleanup:
			if task.(*Task) != nil && task.(*Task).IsCompleted == false {
				task.(*Task).RetryTask()
			}
		default:
		}
	}
}

// Context allows reusing resources betwen Jobs.
type Context struct {
	Buffer *bytes.Buffer
}

// Add adds a job to the queue
func (w *Worker) Add(j interface{}) {
	w.lock.Lock()
	defer w.lock.Unlock()
	w.queue.PushBack(j)

	cnt := w.queue.Len()
	if cnt > w.maxJobs {
		 fmt.Errorf("queue len crossed and len of queue is %d", cnt)
	}
}

// Status returns the number of items in the worker queue
func (w *Worker) Status() int {
	w.lock.Lock()
	defer w.lock.Unlock()
	return w.queue.Len()
}

// Stop the worker
func (w *Worker) Stop() {
	close(w.stop)
	<-w.done
}

// returns job from queue
func (w *Worker) pop() *Task {
	w.lock.Lock()
	defer w.lock.Unlock()

	e := w.queue.Front()
	if e == nil {
		return nil
	}
	w.queue.Remove(e)

	if e.Value == nil {
		return nil
	}

	j := e.Value.(*Task)
	return j
}

func main() {

	var wg sync.WaitGroup
	worker := NewWorker(1)

	//execute the queue
	go worker.Execute()

	//execute the queue
	go worker.Cleanup()


	//Add tasks
	for i := 0; i < 50; i++ {
		wg.Add(1)
		item := &Task{ID: fmt.Sprintf("%d", i), IsCompleted: false, CreationTime: time.Now(), Status: UNTOUCHED}
		item.RunTask = func() {
			//We are making item 24,29 to fail and to append at the end of the queue.
			if item.ID == "24" || item.ID == "29" {
				item.Status = FAILED
				item.IsCompleted = false
			} else {
				item.Status = COMPLETED
				item.IsCompleted = true
				wg.Done()
			}

		}
		item.RetryTask = func() {

			select {
			case <-time.After(1 * time.Second):
				if item.Status == FAILED {
					if time.Now().Sub(item.CreationTime).Seconds() < 15 {
						//TODO:: retry the item.
						worker.queue.PushBack(item)
						return
					} else {
						item.Status = TIMEOUT
						fmt.Printf("Status Timeout for ID: %q", item.ID)
						wg.Done()
					}
				}
				return
			}
		}

		worker.Add(item)
	}

	wg.Wait()

}
