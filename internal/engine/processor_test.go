package engine

import (
	"context"
	"testing"
	"time"
)

func TestSleepIfNeededAcceptsFractionalSeconds(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	startedAt := time.Now()
	if err := sleepIfNeeded(ctx, 0.05); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	elapsed := time.Since(startedAt)
	if elapsed < 40*time.Millisecond {
		t.Fatalf("expected fractional sleep to wait at least 40ms, got %s", elapsed)
	}
	if elapsed > 300*time.Millisecond {
		t.Fatalf("expected fractional sleep to stay well below 300ms, got %s", elapsed)
	}
}
