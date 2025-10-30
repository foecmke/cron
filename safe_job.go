package cron

import (
	"log"
	"runtime/debug"
)

// SafeJob wraps a Job with panic recovery
type SafeJob struct {
	Job
	Name string
}

func (s SafeJob) Run() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Job '%s' panicked: %v\nStack trace:\n%s", s.Name, r, debug.Stack())
		}
	}()
	s.Job.Run()
}

// wrapJob wraps a job with panic recovery
func wrapJob(job Job, name string) Job {
	return SafeJob{Job: job, Name: name}
}