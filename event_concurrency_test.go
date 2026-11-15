package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestEventConcurrency(t *testing.T) {
	app := application.New(application.Options{
		Name: "TestEventConcurrency",
	})
	if app == nil {
		t.Fatal("Failed to create application instance")
	}

	var receivedCount int64
	doneChan := make(chan struct{})

	// Subscribe to event
	app.Event.On("test:concurrent", func(ev *application.CustomEvent) {
		atomic.AddInt64(&receivedCount, 1)
	})

	// Spin up 10 goroutines emitting 50 events each (total 500)
	var wg sync.WaitGroup
	const workers = 10
	const eventsPerWorker = 50
	const totalExpected = workers * eventsPerWorker

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < eventsPerWorker; j++ {
				payload := map[string]string{
					"worker":  fmt.Sprintf("w-%d", workerID),
					"message": fmt.Sprintf("msg-%d", j),
				}
				app.Event.Emit("test:concurrent", payload)
			}
		}(i)
	}

	wg.Wait()

	go func() {
		for {
			if atomic.LoadInt64(&receivedCount) >= int64(totalExpected) {
				close(doneChan)
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()

	select {
	case <-doneChan:
		t.Logf("PASSED: Successfully emitted and processed %d concurrent events across %d workers!", totalExpected, workers)
	case <-time.After(3 * time.Second):
		t.Fatalf("Timeout waiting for events! Received %d of %d", atomic.LoadInt64(&receivedCount), totalExpected)
	}
}
