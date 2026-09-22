package engine

import (
	"context"
	"errors"
	"testing"
)

func TestNewWorkerPool(t *testing.T) {
	wp := NewWorkerPool(5, 10)

	if wp == nil {
		t.Fatal("expected non-nil worker pool")
	}

	if wp.workerCount != 5 {
		t.Fatalf("expected workerCount 5, got %d", wp.workerCount)
	}

	if cap(wp.jobs) != 10 {
		t.Fatalf("expected jobs channel capacity 10, got %d", cap(wp.jobs))
	}
}

func TestWorkerPool_SubmitAndExecute(t *testing.T) {
	wp := NewWorkerPool(2, 5)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wp.Start(ctx)

	done := make(chan struct{}, 10)

	for i := 0; i < 10; i++ {
		err := wp.Submit(ctx, func() {
			done <- struct{}{}
		})
		if err != nil {
			t.Fatalf("submit failed: %v", err)
		}
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	cancel()
	wp.Stop()
}

func TestWorkerPool_SubmitCanceledContext(t *testing.T) {
	wp := NewWorkerPool(1, 1)

	ctx, cancel := context.WithCancel(context.Background())

	_ = wp.Submit(ctx, func() {})

	cancel()

	err := wp.Submit(ctx, func() {})

	if err == nil {
		t.Fatal("expected error on canceled context, got nil")
	}

	if err != context.Canceled {
		t.Fatalf("expected context.Canceled error, got %v", err)
	}
}

func TestWorkerPool_Stop(t *testing.T) {
	wp := NewWorkerPool(3, 10)

	ctx, cancel := context.WithCancel(context.Background())
	wp.Start(ctx)

	cancel()
	wp.Stop()

	err := wp.Submit(context.Background(), func() {})
	if !errors.Is(err, ErrWorkerPoolStopped) {
		t.Fatalf("expected ErrWorkerPoolStopped, got %v", err)
	}
}

func TestWorkerPool_MultipleWorkers(t *testing.T) {
	wp := NewWorkerPool(5, 20)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wp.Start(ctx)

	done := make(chan struct{}, 20)

	for i := 0; i < 20; i++ {
		err := wp.Submit(ctx, func() {
			done <- struct{}{}
		})
		if err != nil {
			t.Fatalf("submit failed: %v", err)
		}
	}

	for i := 0; i < 20; i++ {
		<-done
	}

	cancel()
	wp.Stop()
}

func TestWorkerPool_ConcurrentJobExecution(t *testing.T) {
	wp := NewWorkerPool(3, 10)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wp.Start(ctx)

	started := make(chan struct{}, 6)
	release := make(chan struct{})
	done := make(chan struct{}, 6)

	for i := 0; i < 6; i++ {
		err := wp.Submit(ctx, func() {
			started <- struct{}{}
			<-release
			done <- struct{}{}
		})
		if err != nil {
			t.Fatalf("submit failed: %v", err)
		}
	}

	for i := 0; i < 3; i++ {
		<-started
	}
	close(release)

	for i := 0; i < 6; i++ {
		<-done
	}

	cancel()
	wp.Stop()
}
