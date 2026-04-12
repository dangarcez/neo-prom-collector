package scheduler

import (
	"context"
	"sync"
	"testing"
)

func TestWorkerPoolRunProcessesAllItems(t *testing.T) {
	pool := NewWorkerPool(3)

	visited := make([]bool, 5)
	var mu sync.Mutex

	err := pool.Run(context.Background(), len(visited), func(_ context.Context, index int) error {
		mu.Lock()
		visited[index] = true
		mu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	for index, wasVisited := range visited {
		if !wasVisited {
			t.Fatalf("expected item %d to be processed", index)
		}
	}
}
