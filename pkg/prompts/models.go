package prompts

import (
	"time"
)

// ExtractedEntity represents an entity extracted from content
type ExtractedEntity struct {
	Name         string `json:"entity" mapstructure:"entity" csv:"entity"`
	EntityTypeID int    `json:"entity_type_id" mapstructure:"entity_type_id" csv:"entity_type_id"`
}

// ExtractedEntities represents a list of extracted entities
type ExtractedEntities struct {
	ExtractedEntities []ExtractedEntity `json:"entities"`
}

// MissedEntities represents entities that weren't extracted
type MissedEntities struct {
	MissedEntities []string `json:"missed_entities"`
}

// EntityClassificationTriple represents an entity with classification
type EntityClassificationTriple struct {
	UUID       string  `json:"uuid"`
	Name       string  `json:"name"`
	EntityType *string `json:"entity_type"`
}

// EntityClassification represents entity classifications
type EntityClassification struct {
	EntityClassifications []EntityClassificationTriple `json:"entity_classifications"`
}

// EntitySummary represents an entity summary
type EntitySummary struct {
	Summary string `json:"summary"`
}
type ExtractedEdge struct {
	Name      string    `json:"name" mapstructure:"name" csv:"name"` // matches Python name
	Fact      string    `json:"fact" mapstructure:"fact" csv:"fact"`
	SourceID  int       `json:"source_id" mapstructure:"source_id" csv:"source_id"` // alias for SourceNodeID uuid
	TargetID  int       `json:"target_id" mapstructure:"target_id" csv:"target_id"` // alias for TargetNodeID uuid
	UpdatedAt time.Time `json:"updated_at" mapstructure:"updated_at" csv:"updated_at"`
	Summary   string    `json:"summary,omitempty" mapstructure:"summary" csv:"summary"`
	ValidAt   string    `json:"valid_at,omitempty" mapstructure:"valid_at" csv:"valid_at"`       // matches Python valid_at
	InvalidAt string    `json:"invalid_at,omitempty" mapstructure:"invalid_at" csv:"invalid_at"` // matches Python invalid_at
	// alias for Fact
}

// ExtractedEdges represents a list of extracted edges
type ExtractedEdges struct {
	Edges []ExtractedEdge `json:"facts"`
}

// MissingFacts represents facts that weren't extracted
type MissingFacts struct {
	MissingFacts []string `json:"missing_facts"`
}

// NodeDuplicate represents a node duplicate resolution
type NodeDuplicate struct {
	ID           int    `json:"id" mapstructure:"id" csv:"id"`
	DuplicateIdx int    `json:"duplicate_idx" mapstructure:"duplicate_idx" csv:"duplicate_idx"`
	Name         string `json:"name" mapstructure:"name" csv:"name"`
	Duplicates   []int  `json:"duplicates" mapstructure:"duplicates" csv:"duplicates"`
}

// NodeResolutions represents node duplicate resolutions
type NodeResolutions struct {
	EntityResolutions []NodeDuplicate `json:"entity_resolutions"`
}

// EdgeDuplicate represents edge duplicate detection result
type EdgeDuplicate struct {
	DuplicateFacts    []int  `json:"duplicate_facts"`
	ContradictedFacts []int  `json:"contradicted_facts"`
	FactType          string `json:"fact_type"`
}

// UniqueFact represents a unique fact
type UniqueFact struct {
	UUID string `json:"uuid"`
	Fact string `json:"fact"`
}

// UniqueFacts represents a list of unique facts
type UniqueFacts struct {
	UniqueFacts []UniqueFact `json:"unique_facts"`
}

// InvalidatedEdges represents edges to be invalidated
type InvalidatedEdges struct {
	ContradictedFacts []int `json:"contradicted_facts"`
}

// EdgeDates represents temporal information for edges
type EdgeDates struct {
	ValidAt   *string `json:"valid_at"`
	InvalidAt *string `json:"invalid_at"`
}

// Summary represents a text summary
type Summary struct {
	Summary string `json:"summary"`
}

// SummaryDescription represents a summary description
type SummaryDescription struct {
	Description string `json:"description"`
}

// QueryExpansion represents an expanded query
type QueryExpansion struct {
	Query string `json:"query"`
}

// QAResponse represents a question-answer response
type QAResponse struct {
	Answer string `json:"ANSWER"`
}

// EvalResponse represents an evaluation response
type EvalResponse struct {
	IsCorrect bool   `json:"is_correct"`
	Reasoning string `json:"reasoning"`
}

// EvalAddEpisodeResults represents evaluation of episode addition results
type EvalAddEpisodeResults struct {
	CandidateIsWorse bool   `json:"candidate_is_worse"`
	Reasoning        string `json:"reasoning"`
}

// Episode represents an episode context for prompts
type Episode struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Content   string                 `json:"content"`
	Reference time.Time              `json:"reference"`
	CreatedAt time.Time              `json:"created_at"`
	GroupID   string                 `json:"group_id"`
	Metadata  map[string]interface{} `json:"metadata"`
}
