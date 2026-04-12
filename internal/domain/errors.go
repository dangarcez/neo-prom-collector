package domain

import "errors"

var (
	ErrAmbiguousNodeMatch = errors.New("ambiguous node match")
	ErrSourceNodeMissing  = errors.New("source node not found")
	ErrTargetNodeMissing  = errors.New("target node not found")
)
