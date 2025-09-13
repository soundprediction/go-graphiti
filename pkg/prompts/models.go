package prompts

import "time"

// ExtractedEntity represents an entity extracted from text.
type ExtractedEntity struct {
	Name         string `json:"name" description:"Name of the extracted entity"`
	EntityTypeID int    `json:"entity_type_id" description:"ID of the classified entity type. Must be one of the provided entity_type_id integers."`
}

// ExtractedEntities is a collection of extracted entities.
type ExtractedEntities struct {
	ExtractedEntities []ExtractedEntity `json:"extracted_entities" description:"List of extracted entities"`
}

// MissedEntities represents entities that weren't extracted.
type MissedEntities struct {
	MissedEntities []string `json:"missed_entities" description:"Names of entities that weren't extracted"`
}

// EntityClassificationTriple represents a classified entity.
type EntityClassificationTriple struct {
	UUID       string  `json:"uuid" description:"UUID of the entity"`
	Name       string  `json:"name" description:"Name of the entity"`
	EntityType *string `json:"entity_type" description:"Type of the entity. Must be one of the provided types or None"`
}

// EntityClassification is a collection of entity classifications.
type EntityClassification struct {
	EntityClassifications []EntityClassificationTriple `json:"entity_classifications" description:"List of entity classifications"`
}

// Edge represents a relationship between entities.
type Edge struct {
	RelationType     string     `json:"relation_type" description:"FACT_PREDICATE_IN_SCREAMING_SNAKE_CASE"`
	SourceEntityID   int        `json:"source_entity_id" description:"The id of the source entity of the fact"`
	TargetEntityID   int        `json:"target_entity_id" description:"The id of the target entity of the fact"`
	Fact             string     `json:"fact" description:"The fact description"`
	ValidAt          *time.Time `json:"valid_at,omitempty" description:"The date and time when the relationship described by the edge fact became true or was established. Use ISO 8601 format"`
	InvalidAt        *time.Time `json:"invalid_at,omitempty" description:"The date and time when the relationship described by the edge fact stopped being true or ended. Use ISO 8601 format"`
}

// ExtractedEdges is a collection of extracted edges.
type ExtractedEdges struct {
	Edges []Edge `json:"edges" description:"List of extracted edges"`
}

// MissingFacts represents facts that weren't extracted.
type MissingFacts struct {
	MissingFacts []string `json:"missing_facts" description:"facts that weren't extracted"`
}

// NodeDuplicate represents a duplicate node resolution.
type NodeDuplicate struct {
	ID           int    `json:"id" description:"integer id of the entity"`
	DuplicateIdx int    `json:"duplicate_idx" description:"idx of the duplicate entity. If no duplicate entities are found, default to -1"`
	Name         string `json:"name" description:"Name of the entity. Should be the most complete and descriptive name of the entity. Do not include any JSON formatting in the Entity name such as {}"`
	Duplicates   []int  `json:"duplicates" description:"idx of all entities that are a duplicate of the entity with the above id"`
}

// NodeResolutions is a collection of resolved nodes.
type NodeResolutions struct {
	EntityResolutions []NodeDuplicate `json:"entity_resolutions" description:"List of resolved nodes"`
}

// EdgeDuplicate represents a duplicate edge resolution.
type EdgeDuplicate struct {
	ID           int   `json:"id" description:"integer id of the edge"`
	DuplicateIdx int   `json:"duplicate_idx" description:"idx of the duplicate edge. If no duplicate edges are found, default to -1"`
	Duplicates   []int `json:"duplicates" description:"idx of all edges that are a duplicate of the edge with the above id"`
}

// EdgeResolutions is a collection of resolved edges.
type EdgeResolutions struct {
	EdgeResolutions []EdgeDuplicate `json:"edge_resolutions" description:"List of resolved edges"`
}

// EdgeInvalidation represents invalidated edges.
type EdgeInvalidation struct {
	InvalidatedEdgeIDs []int `json:"invalidated_edge_ids" description:"List of edge IDs to invalidate"`
}

// EdgeDateExtraction represents extracted edge dates.
type EdgeDateExtraction struct {
	EdgeID    int        `json:"edge_id" description:"ID of the edge"`
	ValidAt   *time.Time `json:"valid_at,omitempty" description:"When the edge became valid"`
	InvalidAt *time.Time `json:"invalid_at,omitempty" description:"When the edge became invalid"`
}

// ExtractedEdgeDates is a collection of extracted edge dates.
type ExtractedEdgeDates struct {
	ExtractedDates []EdgeDateExtraction `json:"extracted_dates" description:"List of extracted edge dates"`
}

// NodeSummary represents a summarized node.
type NodeSummary struct {
	Summary string `json:"summary" description:"Summary of the node"`
}

// EvalResult represents evaluation results.
type EvalResult struct {
	Score       float64 `json:"score" description:"Evaluation score"`
	Explanation string  `json:"explanation" description:"Explanation of the evaluation"`
}