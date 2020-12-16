package main

import (
	"time"
)

type Task struct {
	ID           string
	IsCompleted  bool
	Status       Status    // untouched, completed, failed, timeout
	CreationTime time.Time // when was the task created
	TaskData     string    // field containing data about the task
	RunTask      RunTask
	RetryTask    RetryTask
}

type RunTask func()
type RetryTask func()

//Status defines the status of the task
type Status string

const (
	UNTOUCHED Status = "untouched"
	COMPLETED Status = "completed"
	FAILED    Status = "failed"
	TIMEOUT   Status = "timeout"
)
