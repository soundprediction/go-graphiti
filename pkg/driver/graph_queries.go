package driver

import (
	"fmt"
	"strings"
)

// GraphProvider and constants are defined in driver.go

// Mapping from Neo4j fulltext index names to FalkorDB node labels
var neo4jToFalkorDBMapping = map[string]string{
	"node_name_and_summary": "Entity",
	"community_name":        "Community",
	"episode_content":       "Episodic",
	"edge_name_and_fact":    "RELATES_TO",
}

// Mapping from fulltext index names to Kuzu node labels
var indexToLabelKuzuMapping = map[string]string{
	"node_name_and_summary": "Entity",
	"community_name":        "Community",
	"episode_content":       "Episodic",
	"edge_name_and_fact":    "RelatesToNode_",
}

// GetRangeIndices returns database-specific range index creation queries
func GetRangeIndices(provider GraphProvider) []string {
	switch provider {
	case GraphProviderFalkorDB:
		return []string{
			// Entity node
			"CREATE INDEX FOR (n:Entity) ON (n.uuid, n.group_id, n.name, n.created_at)",
			// Episodic node
			"CREATE INDEX FOR (n:Episodic) ON (n.uuid, n.group_id, n.created_at, n.valid_at)",
			// Community node
			"CREATE INDEX FOR (n:Community) ON (n.uuid)",
			// RELATES_TO edge
			"CREATE INDEX FOR ()-[e:RELATES_TO]-() ON (e.uuid, e.group_id, e.name, e.created_at, e.expired_at, e.valid_at, e.invalid_at)",
			// MENTIONS edge
			"CREATE INDEX FOR ()-[e:MENTIONS]-() ON (e.uuid, e.group_id)",
			// HAS_MEMBER edge
			"CREATE INDEX FOR ()-[e:HAS_MEMBER]-() ON (e.uuid)",
		}

	case GraphProviderKuzu:
		return []string{} // Kuzu doesn't require explicit range index creation

	default: // Neo4j
		return []string{
			"CREATE INDEX entity_uuid IF NOT EXISTS FOR (n:Entity) ON (n.uuid)",
			"CREATE INDEX episode_uuid IF NOT EXISTS FOR (n:Episodic) ON (n.uuid)",
			"CREATE INDEX community_uuid IF NOT EXISTS FOR (n:Community) ON (n.uuid)",
			"CREATE INDEX relation_uuid IF NOT EXISTS FOR ()-[e:RELATES_TO]-() ON (e.uuid)",
			"CREATE INDEX mention_uuid IF NOT EXISTS FOR ()-[e:MENTIONS]-() ON (e.uuid)",
			"CREATE INDEX has_member_uuid IF NOT EXISTS FOR ()-[e:HAS_MEMBER]-() ON (e.uuid)",
			"CREATE INDEX entity_group_id IF NOT EXISTS FOR (n:Entity) ON (n.group_id)",
			"CREATE INDEX episode_group_id IF NOT EXISTS FOR (n:Episodic) ON (n.group_id)",
			"CREATE INDEX community_group_id IF NOT EXISTS FOR (n:Community) ON (n.group_id)",
			"CREATE INDEX relation_group_id IF NOT EXISTS FOR ()-[e:RELATES_TO]-() ON (e.group_id)",
			"CREATE INDEX mention_group_id IF NOT EXISTS FOR ()-[e:MENTIONS]-() ON (e.group_id)",
			"CREATE INDEX name_entity_index IF NOT EXISTS FOR (n:Entity) ON (n.name)",
			"CREATE INDEX created_at_entity_index IF NOT EXISTS FOR (n:Entity) ON (n.created_at)",
			"CREATE INDEX created_at_episodic_index IF NOT EXISTS FOR (n:Episodic) ON (n.created_at)",
			"CREATE INDEX valid_at_episodic_index IF NOT EXISTS FOR (n:Episodic) ON (n.valid_at)",
			"CREATE INDEX name_edge_index IF NOT EXISTS FOR ()-[e:RELATES_TO]-() ON (e.name)",
			"CREATE INDEX created_at_edge_index IF NOT EXISTS FOR ()-[e:RELATES_TO]-() ON (e.created_at)",
			"CREATE INDEX expired_at_edge_index IF NOT EXISTS FOR ()-[e:RELATES_TO]-() ON (e.expired_at)",
			"CREATE INDEX valid_at_edge_index IF NOT EXISTS FOR ()-[e:RELATES_TO]-() ON (e.valid_at)",
			"CREATE INDEX invalid_at_edge_index IF NOT EXISTS FOR ()-[e:RELATES_TO]-() ON (e.invalid_at)",
		}
	}
}

// GetFulltextIndices returns database-specific fulltext index creation queries
func GetFulltextIndices(provider GraphProvider) []string {
	switch provider {
	case GraphProviderFalkorDB:
		return []string{
			"CREATE FULLTEXT INDEX FOR (e:Episodic) ON (e.content, e.source, e.source_description, e.group_id)",
			"CREATE FULLTEXT INDEX FOR (n:Entity) ON (n.name, n.summary, n.group_id)",
			"CREATE FULLTEXT INDEX FOR (n:Community) ON (n.name, n.group_id)",
			"CREATE FULLTEXT INDEX FOR ()-[e:RELATES_TO]-() ON (e.name, e.fact, e.group_id)",
		}

	case GraphProviderKuzu:
		return []string{
			"CALL CREATE_FTS_INDEX('Episodic', 'episode_content', ['content', 'source', 'source_description']);",
			"CALL CREATE_FTS_INDEX('Entity', 'node_name_and_summary', ['name', 'summary']);",
			"CALL CREATE_FTS_INDEX('Community', 'community_name', ['name']);",
			"CALL CREATE_FTS_INDEX('RelatesToNode_', 'edge_name_and_fact', ['name', 'fact']);",
		}

	default: // Neo4j
		return []string{
			`CREATE FULLTEXT INDEX episode_content IF NOT EXISTS
FOR (e:Episodic) ON EACH [e.content, e.source, e.source_description, e.group_id]`,
			`CREATE FULLTEXT INDEX node_name_and_summary IF NOT EXISTS
FOR (n:Entity) ON EACH [n.name, n.summary, n.group_id]`,
			`CREATE FULLTEXT INDEX community_name IF NOT EXISTS
FOR (n:Community) ON EACH [n.name, n.group_id]`,
			`CREATE FULLTEXT INDEX edge_name_and_fact IF NOT EXISTS
FOR ()-[e:RELATES_TO]-() ON EACH [e.name, e.fact, e.group_id]`,
		}
	}
}

// GetNodesQuery returns database-specific fulltext search query for nodes
func GetNodesQuery(indexName, query string, limit int, provider GraphProvider) string {
	switch provider {
	case GraphProviderFalkorDB:
		label := neo4jToFalkorDBMapping[indexName]
		return fmt.Sprintf("CALL db.idx.fulltext.queryNodes('%s', %s)", label, query)

	case GraphProviderKuzu:
		label := indexToLabelKuzuMapping[indexName]
		return fmt.Sprintf("CALL QUERY_FTS_INDEX('%s', '%s', %s, TOP := $limit)", label, indexName, query)

	default: // Neo4j
		return fmt.Sprintf(`CALL db.index.fulltext.queryNodes("%s", %s, {limit: $limit})`, indexName, query)
	}
}

// GetRelationshipsQuery returns database-specific fulltext search query for relationships
func GetRelationshipsQuery(indexName string, limit int, provider GraphProvider) string {
	switch provider {
	case GraphProviderFalkorDB:
		label := neo4jToFalkorDBMapping[indexName]
		return fmt.Sprintf("CALL db.idx.fulltext.queryRelationships('%s', $query)", label)

	case GraphProviderKuzu:
		label := indexToLabelKuzuMapping[indexName]
		return fmt.Sprintf("CALL QUERY_FTS_INDEX('%s', '%s', cast($query AS STRING), TOP := $limit)", label, indexName)

	default: // Neo4j
		return fmt.Sprintf(`CALL db.index.fulltext.queryRelationships("%s", $query, {limit: $limit})`, indexName)
	}
}

// GetVectorCosineFuncQuery returns database-specific cosine similarity function query
func GetVectorCosineFuncQuery(vec1, vec2 string, provider GraphProvider) string {
	switch provider {
	case GraphProviderFalkorDB:
		// FalkorDB uses a different syntax for regular cosine similarity and Neo4j uses normalized cosine similarity
		return fmt.Sprintf("(2 - vec.cosineDistance(%s, vecf32(%s)))/2", vec1, vec2)

	case GraphProviderKuzu:
		return fmt.Sprintf("array_cosine_similarity(%s, %s)", vec1, vec2)

	default: // Neo4j
		return fmt.Sprintf("vector.similarity.cosine(%s, %s)", vec1, vec2)
	}
}

// QueryBuilder provides database-agnostic query building utilities
type QueryBuilder struct {
	provider GraphProvider
}

// NewQueryBuilder creates a new query builder for the specified provider
func NewQueryBuilder(provider GraphProvider) *QueryBuilder {
	return &QueryBuilder{
		provider: provider,
	}
}

// BuildFulltextNodeQuery builds a fulltext search query for nodes
func (qb *QueryBuilder) BuildFulltextNodeQuery(indexName, searchTerm string, limit int) string {
	return GetNodesQuery(indexName, searchTerm, limit, qb.provider)
}

// BuildFulltextRelationshipQuery builds a fulltext search query for relationships
func (qb *QueryBuilder) BuildFulltextRelationshipQuery(indexName string, limit int) string {
	return GetRelationshipsQuery(indexName, limit, qb.provider)
}

// BuildCosineSimilarityQuery builds a cosine similarity query
func (qb *QueryBuilder) BuildCosineSimilarityQuery(vec1, vec2 string) string {
	return GetVectorCosineFuncQuery(vec1, vec2, qb.provider)
}

// GetRangeIndexQueries returns all range index creation queries for this provider
func (qb *QueryBuilder) GetRangeIndexQueries() []string {
	return GetRangeIndices(qb.provider)
}

// GetFulltextIndexQueries returns all fulltext index creation queries for this provider
func (qb *QueryBuilder) GetFulltextIndexQueries() []string {
	return GetFulltextIndices(qb.provider)
}

// GetProvider returns the current graph provider
func (qb *QueryBuilder) GetProvider() GraphProvider {
	return qb.provider
}

// SetProvider sets the graph provider
func (qb *QueryBuilder) SetProvider(provider GraphProvider) {
	qb.provider = provider
}

// EscapeQueryString escapes special characters in search queries
func EscapeQueryString(query string) string {
	// Escape special characters for fulltext search
	replacer := strings.NewReplacer(
		`"`, `\"`,
		`\`, `\\`,
		`+`, `\+`,
		`-`, `\-`,
		`!`, `\!`,
		`(`, `\(`,
		`)`, `\)`,
		`{`, `\{`,
		`}`, `\}`,
		`[`, `\[`,
		`]`, `\]`,
		`^`, `\^`,
		`~`, `\~`,
		`*`, `\*`,
		`?`, `\?`,
		`:`, `\:`,
		`|`, `\|`,
		`&`, `\&`,
	)
	return replacer.Replace(query)
}

// BuildParameterizedQuery builds a query with parameter placeholders
func BuildParameterizedQuery(query string, params map[string]interface{}) (string, map[string]interface{}) {
	// Clean parameters by removing internal driver parameters
	cleanParams := make(map[string]interface{})
	for key, value := range params {
		if !strings.HasSuffix(key, "_") && value != nil {
			cleanParams[key] = value
		}
	}

	return query, cleanParams
}
