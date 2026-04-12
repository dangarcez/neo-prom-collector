package observability

import (
	"log/slog"
	"sync/atomic"

	"neo_collector_go/internal/domain"
)

type Metrics struct {
	jobsStarted     atomic.Int64
	jobsCompleted   atomic.Int64
	jobsFailed      atomic.Int64
	jobsSkipped     atomic.Int64
	datapoints      atomic.Int64
	datapointErrors atomic.Int64
	nodesCreated    atomic.Int64
	nodesUpdated    atomic.Int64
	nodesSkipped    atomic.Int64
	relsCreated     atomic.Int64
	relsUpdated     atomic.Int64
	relsSkipped     atomic.Int64
}

type Snapshot struct {
	JobsStarted          int64
	JobsCompleted        int64
	JobsFailed           int64
	JobsSkipped          int64
	Datapoints           int64
	DatapointErrors      int64
	NodesCreated         int64
	NodesUpdated         int64
	NodesSkipped         int64
	RelationshipsCreated int64
	RelationshipsUpdated int64
	RelationshipsSkipped int64
}

func NewMetrics() *Metrics {
	return &Metrics{}
}

func (m *Metrics) RecordJobStarted() {
	if m == nil {
		return
	}
	m.jobsStarted.Add(1)
}

func (m *Metrics) RecordJobSkipped() {
	if m == nil {
		return
	}
	m.jobsSkipped.Add(1)
}

func (m *Metrics) RecordJobFinished(err error) {
	if m == nil {
		return
	}
	if err != nil {
		m.jobsFailed.Add(1)
		return
	}
	m.jobsCompleted.Add(1)
}

func (m *Metrics) RecordProcessStats(stats domain.ProcessStats) {
	if m == nil {
		return
	}

	m.datapoints.Add(int64(stats.Datapoints))
	m.datapointErrors.Add(int64(stats.Errors))
	m.nodesCreated.Add(int64(stats.NodesCreated))
	m.nodesUpdated.Add(int64(stats.NodesUpdated))
	m.nodesSkipped.Add(int64(stats.NodesSkipped))
	m.relsCreated.Add(int64(stats.RelationshipsCreated))
	m.relsUpdated.Add(int64(stats.RelationshipsUpdated))
	m.relsSkipped.Add(int64(stats.RelationshipsSkipped))
}

func (m *Metrics) Snapshot() Snapshot {
	if m == nil {
		return Snapshot{}
	}

	return Snapshot{
		JobsStarted:          m.jobsStarted.Load(),
		JobsCompleted:        m.jobsCompleted.Load(),
		JobsFailed:           m.jobsFailed.Load(),
		JobsSkipped:          m.jobsSkipped.Load(),
		Datapoints:           m.datapoints.Load(),
		DatapointErrors:      m.datapointErrors.Load(),
		NodesCreated:         m.nodesCreated.Load(),
		NodesUpdated:         m.nodesUpdated.Load(),
		NodesSkipped:         m.nodesSkipped.Load(),
		RelationshipsCreated: m.relsCreated.Load(),
		RelationshipsUpdated: m.relsUpdated.Load(),
		RelationshipsSkipped: m.relsSkipped.Load(),
	}
}

func (m *Metrics) LogSnapshot(logger *slog.Logger) {
	if m == nil || logger == nil {
		return
	}

	snapshot := m.Snapshot()
	logger.Info("runtime metrics snapshot",
		"jobs_started", snapshot.JobsStarted,
		"jobs_completed", snapshot.JobsCompleted,
		"jobs_failed", snapshot.JobsFailed,
		"jobs_skipped", snapshot.JobsSkipped,
		"datapoints", snapshot.Datapoints,
		"datapoint_errors", snapshot.DatapointErrors,
		"nodes_created", snapshot.NodesCreated,
		"nodes_updated", snapshot.NodesUpdated,
		"nodes_skipped", snapshot.NodesSkipped,
		"relationships_created", snapshot.RelationshipsCreated,
		"relationships_updated", snapshot.RelationshipsUpdated,
		"relationships_skipped", snapshot.RelationshipsSkipped,
	)
}
