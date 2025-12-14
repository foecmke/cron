package cron

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Test ParseWithError with valid specs
func TestParseWithError_Valid(t *testing.T) {
	specs := []string{
		"* * * * * *",
		"0 0 0 * * *",
		"@hourly",
		"@daily",
		"@every 1h",
	}

	for _, spec := range specs {
		_, err := ParseWithError(spec)
		if err != nil {
			t.Errorf("ParseWithError(%q) returned unexpected error: %v", spec, err)
		}
	}
}

// Test ParseWithError with invalid specs
func TestParseWithError_Invalid(t *testing.T) {
	specs := []string{
		"",
		"* * *",
		"* * * * * * *",
		"invalid",
		"@unknown",
		"@every invalid",
		"70 * * * * *",
		"* 70 * * * *",
		"* * 25 * * *",
		"* * * 32 * *",
		"* * * * 13 *",
		"* * * * * 7",
	}

	for _, spec := range specs {
		_, err := ParseWithError(spec)
		if err == nil {
			t.Errorf("ParseWithError(%q) should return error but got nil", spec)
		}
	}
}

// Test AddFuncWithError with valid job
func TestAddFuncWithError_Valid(t *testing.T) {
	cron := New()
	err := cron.AddFuncWithError("* * * * * *", func() {}, "test_job")
	if err != nil {
		t.Errorf("AddFuncWithError returned unexpected error: %v", err)
	}
}

// Test AddFuncWithError with invalid spec
func TestAddFuncWithError_InvalidSpec(t *testing.T) {
	cron := New()
	err := cron.AddFuncWithError("invalid", func() {}, "test_job")
	if err == nil {
		t.Error("AddFuncWithError should return error for invalid spec")
	}
}

// Test AddFuncWithError with duplicate name
func TestAddFuncWithError_DuplicateName(t *testing.T) {
	cron := New()
	err := cron.AddFuncWithError("* * * * * *", func() {}, "test_job")
	if err != nil {
		t.Fatalf("First AddFuncWithError failed: %v", err)
	}

	err = cron.AddFuncWithError("* * * * * *", func() {}, "test_job")
	if err == nil {
		t.Error("AddFuncWithError should return error for duplicate name")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Error message should mention 'already exists', got: %v", err)
	}
}

// Test AddJobWithError
func TestAddJobWithError(t *testing.T) {
	cron := New()
	job := FuncJob(func() {})

	err := cron.AddJobWithError("* * * * * *", job, "test_job")
	if err != nil {
		t.Errorf("AddJobWithError returned unexpected error: %v", err)
	}

	// Test duplicate
	err = cron.AddJobWithError("* * * * * *", job, "test_job")
	if err == nil {
		t.Error("AddJobWithError should return error for duplicate name")
	}
}

// Test RemoveJobWithResult
func TestRemoveJobWithResult(t *testing.T) {
	cron := New()
	cron.AddFunc("* * * * * *", func() {}, "test_job")

	// Remove existing job
	result := cron.RemoveJobWithResult("test_job")
	if !result {
		t.Error("RemoveJobWithResult should return true for existing job")
	}

	// Remove non-existing job
	result = cron.RemoveJobWithResult("non_existing")
	if result {
		t.Error("RemoveJobWithResult should return false for non-existing job")
	}
}

// Test RemoveJobWithResult while running
func TestRemoveJobWithResult_WhileRunning(t *testing.T) {
	cron := New()
	cron.AddFunc("* * * * * *", func() {}, "test_job")
	cron.Start()
	defer cron.Stop()

	time.Sleep(10 * time.Millisecond)

	result := cron.RemoveJobWithResult("test_job")
	if !result {
		t.Error("RemoveJobWithResult should return true for existing job while running")
	}
}

// Test StopWithTimeout - successful stop
func TestStopWithTimeout_Success(t *testing.T) {
	cron := New()
	cron.Start()

	err := cron.StopWithTimeout(1 * time.Second)
	if err != nil {
		t.Errorf("StopWithTimeout returned unexpected error: %v", err)
	}
}

// Test StopWithTimeout - already stopped
func TestStopWithTimeout_AlreadyStopped(t *testing.T) {
	cron := New()
	err := cron.StopWithTimeout(1 * time.Second)
	if err != nil {
		t.Errorf("StopWithTimeout on non-running cron should not error: %v", err)
	}
}

// Test Stop on already stopped cron
func TestStop_AlreadyStopped(t *testing.T) {
	cron := New()
	cron.Stop() // Should not panic or block
}

// Test StopAndWait
func TestStopAndWait(t *testing.T) {
	cron := New()
	var counter int32
	var wg sync.WaitGroup
	wg.Add(1)

	cron.AddFunc("* * * * * *", func() {
		atomic.AddInt32(&counter, 1)
		wg.Done()
		time.Sleep(100 * time.Millisecond)
	}, "test_job")

	cron.Start()

	// Wait for job to start
	wg.Wait()

	err := cron.StopAndWait(2 * time.Second)
	if err != nil {
		t.Errorf("StopAndWait returned unexpected error: %v", err)
	}

	if atomic.LoadInt32(&counter) == 0 {
		t.Error("Job should have run at least once")
	}
}

// Test GetMetrics
func TestGetMetrics(t *testing.T) {
	cron := New()
	var wg sync.WaitGroup
	wg.Add(2)

	cron.AddFunc("* * * * * *", func() {
		wg.Done()
	}, "test_job")

	cron.Start()
	defer cron.Stop()

	// Wait for job to run twice
	wg.Wait()

	time.Sleep(50 * time.Millisecond)

	metrics := cron.GetMetrics()
	if metrics.TotalRuns < 2 {
		t.Errorf("Expected at least 2 runs, got %d", metrics.TotalRuns)
	}
}

// Test metrics with panic
func TestMetrics_WithPanic(t *testing.T) {
	cron := New()
	var wg sync.WaitGroup
	var panicCount int32
	wg.Add(1)

	cron.AddFunc("* * * * * *", func() {
		if atomic.AddInt32(&panicCount, 1) == 1 {
			wg.Done()
		}
		panic("test panic")
	}, "panic_job")

	cron.Start()
	defer cron.Stop()

	wg.Wait()
	time.Sleep(50 * time.Millisecond)

	metrics := cron.GetMetrics()
	if metrics.TotalPanics == 0 {
		t.Error("Expected at least 1 panic")
	}
	if metrics.TotalRuns == 0 {
		t.Error("Expected at least 1 run")
	}

	// Remove the job to stop further panics
	cron.RemoveJob("panic_job")
	time.Sleep(50 * time.Millisecond)
}

// Test metrics active jobs
func TestMetrics_ActiveJobs(t *testing.T) {
	cron := New()
	var started sync.WaitGroup
	var finish sync.WaitGroup
	started.Add(1)
	finish.Add(1)

	cron.AddFunc("* * * * * *", func() {
		started.Done()
		finish.Wait()
	}, "long_job")

	cron.Start()
	defer cron.Stop()

	started.Wait()
	time.Sleep(10 * time.Millisecond)

	metrics := cron.GetMetrics()
	if metrics.ActiveJobs == 0 {
		t.Error("Expected at least 1 active job")
	}

	finish.Done()
	time.Sleep(50 * time.Millisecond)

	metrics = cron.GetMetrics()
	if metrics.ActiveJobs != 0 {
		t.Errorf("Expected 0 active jobs, got %d", metrics.ActiveJobs)
	}
}

// Test backward compatibility - original AddFunc still works
func TestBackwardCompatibility_AddFunc(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	cron := New()
	cron.AddFunc("* * * * * ?", func() { wg.Done() }, "test_job")
	cron.Start()
	defer cron.Stop()

	select {
	case <-time.After(ONE_SECOND):
		t.FailNow()
	case <-wait(wg):
	}
}

// Test backward compatibility - original AddJob still works
func TestBackwardCompatibility_AddJob(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	cron := New()
	cron.AddJob("* * * * * ?", testJob{wg, "job"}, "test_job")
	cron.Start()
	defer cron.Stop()

	select {
	case <-time.After(ONE_SECOND):
		t.FailNow()
	case <-wait(wg):
	}
}

// Test backward compatibility - original RemoveJob still works
func TestBackwardCompatibility_RemoveJob(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)

	cron := New()
	cron.AddFunc("* * * * * ?", func() { wg.Done() }, "test_job")
	cron.RemoveJob("test_job")
	cron.Start()
	defer cron.Stop()

	select {
	case <-time.After(ONE_SECOND):
		// Job should not run
	case <-wait(wg):
		t.FailNow()
	}
}

// Test backward compatibility - original Parse still works
func TestBackwardCompatibility_Parse(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Parse should not panic on valid spec: %v", r)
		}
	}()

	schedule := Parse("* * * * * *")
	if schedule == nil {
		t.Error("Parse returned nil for valid spec")
	}
}

// Test backward compatibility - Parse panics on invalid spec
func TestBackwardCompatibility_Parse_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Parse should panic on invalid spec")
		}
	}()

	Parse("invalid")
}

// Test concurrent AddFuncWithError
func TestConcurrent_AddFuncWithError(t *testing.T) {
	cron := New()
	var wg sync.WaitGroup
	errors := make([]error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			err := cron.AddFuncWithError("* * * * * *", func() {}, "test_job")
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	successCount := 0
	for _, err := range errors {
		if err == nil {
			successCount++
		}
	}

	if successCount != 1 {
		t.Errorf("Expected exactly 1 success, got %d", successCount)
	}
}

// Test StopAndWait timeout
func TestStopAndWait_Timeout(t *testing.T) {
	cron := New()
	var started sync.WaitGroup
	started.Add(1)

	cron.AddFunc("* * * * * *", func() {
		started.Done()
		time.Sleep(5 * time.Second) // Long running job
	}, "long_job")

	cron.Start()
	started.Wait()

	err := cron.StopAndWait(100 * time.Millisecond)
	if err == nil {
		t.Error("StopAndWait should timeout")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("Error should mention timeout, got: %v", err)
	}
}

// Test multiple stops don't cause issues
func TestMultipleStops(t *testing.T) {
	cron := New()
	cron.Start()
	cron.Stop()
	cron.Stop() // Should not panic or block
	cron.Stop() // Should not panic or block
}

// Test AddFuncWithError while running
func TestAddFuncWithError_WhileRunning(t *testing.T) {
	cron := New()
	cron.Start()
	defer cron.Stop()

	var wg sync.WaitGroup
	wg.Add(1)

	err := cron.AddFuncWithError("* * * * * *", func() { wg.Done() }, "test_job")
	if err != nil {
		t.Errorf("AddFuncWithError while running returned error: %v", err)
	}

	select {
	case <-time.After(ONE_SECOND):
		t.Error("Job should have run")
	case <-wait(&wg):
	}
}
