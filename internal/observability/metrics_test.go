package observability

import (
	"errors"
	"testing"

	"neo_collector_go/internal/domain"
)

func TestMetricsSnapshot(t *testing.T) {
	metrics := NewMetrics()

	metrics.RecordJobStarted()
	metrics.RecordJobStarted()
	metrics.RecordJobFinished(nil)
	metrics.RecordJobFinished(errors.New("boom"))
	metrics.RecordJobSkipped()
	metrics.RecordProcessStats(domain.ProcessStats{
		Datapoints: 2,
		Errors:     1,
		ApplyStats: domain.ApplyStats{
			NodesCreated:         1,
			NodesUpdated:         1,
			NodesSkipped:         1,
			RelationshipsCreated: 1,
			RelationshipsUpdated: 0,
			RelationshipsSkipped: 1,
		},
	})

	snapshot := metrics.Snapshot()
	if snapshot.JobsStarted != 2 {
		t.Fatalf("expected 2 jobs started, got %d", snapshot.JobsStarted)
	}
	if snapshot.JobsCompleted != 1 {
		t.Fatalf("expected 1 job completed, got %d", snapshot.JobsCompleted)
	}
	if snapshot.JobsFailed != 1 {
		t.Fatalf("expected 1 job failed, got %d", snapshot.JobsFailed)
	}
	if snapshot.JobsSkipped != 1 {
		t.Fatalf("expected 1 job skipped, got %d", snapshot.JobsSkipped)
	}
	if snapshot.Datapoints != 2 || snapshot.DatapointErrors != 1 {
		t.Fatalf("unexpected datapoint metrics: %#v", snapshot)
	}
	if snapshot.NodesCreated != 1 || snapshot.NodesUpdated != 1 || snapshot.NodesSkipped != 1 {
		t.Fatalf("unexpected node metrics: %#v", snapshot)
	}
	if snapshot.RelationshipsCreated != 1 || snapshot.RelationshipsSkipped != 1 {
		t.Fatalf("unexpected relationship metrics: %#v", snapshot)
	}
}
