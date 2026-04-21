package domain

type NodeSelector struct {
	Type       string
	Attributes map[string]any
}

type GraphNode struct {
	Types             []string
	Name              string
	TemplateHashes    []string
	UpdatePolicy      UpdatePolicy
	ExpirationTimeMin *int
	Properties        map[string]any
	UID               string
}

type GraphRelationship struct {
	Type              string
	TemplateHash      string
	UpdatePolicy      UpdatePolicy
	ExpirationTimeMin *int
	Source            NodeSelector
	Target            NodeSelector
	Properties        map[string]any
	UID               string
}

type MutationPlan struct {
	Nodes         []GraphNode
	Relationships []GraphRelationship
}

type PersistAction string

const (
	PersistActionCreated PersistAction = "created"
	PersistActionUpdated PersistAction = "updated"
	PersistActionSkipped PersistAction = "skipped"
)

type ApplyStats struct {
	NodesCreated         int
	NodesUpdated         int
	NodesSkipped         int
	RelationshipsCreated int
	RelationshipsUpdated int
	RelationshipsSkipped int
}

func (s *ApplyStats) AddNode(action PersistAction) {
	switch action {
	case PersistActionCreated:
		s.NodesCreated++
	case PersistActionUpdated:
		s.NodesUpdated++
	case PersistActionSkipped:
		s.NodesSkipped++
	}
}

func (s *ApplyStats) AddRelationship(action PersistAction) {
	switch action {
	case PersistActionCreated:
		s.RelationshipsCreated++
	case PersistActionUpdated:
		s.RelationshipsUpdated++
	case PersistActionSkipped:
		s.RelationshipsSkipped++
	}
}

func (s *ApplyStats) Merge(other ApplyStats) {
	s.NodesCreated += other.NodesCreated
	s.NodesUpdated += other.NodesUpdated
	s.NodesSkipped += other.NodesSkipped
	s.RelationshipsCreated += other.RelationshipsCreated
	s.RelationshipsUpdated += other.RelationshipsUpdated
	s.RelationshipsSkipped += other.RelationshipsSkipped
}

type ProcessStats struct {
	Datapoints int
	Errors     int
	ApplyStats
}
