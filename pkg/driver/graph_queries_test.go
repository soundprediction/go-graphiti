package driver

import (
	"strings"
	"testing"
)

func TestGraphProvider(t *testing.T) {
	providers := []GraphProvider{
		GraphProviderNeo4j,
		GraphProviderFalkorDB,
		GraphProviderKuzu,
	}

	for _, provider := range providers {
		t.Run(string(provider), func(t *testing.T) {
			// Test that provider constants are defined
			if string(provider) == "" {
				t.Errorf("Provider %s should not be empty", provider)
			}
		})
	}
}

func TestGetRangeIndices(t *testing.T) {
	tests := []struct {
		provider GraphProvider
		expected int // expected number of indices
	}{
		{GraphProviderNeo4j, 20},     // Neo4j has 20 range indices
		{GraphProviderFalkorDB, 6},   // FalkorDB has 6 range indices
		{GraphProviderKuzu, 0},       // Kuzu has 0 range indices
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			indices := GetRangeIndices(tt.provider)
			if len(indices) != tt.expected {
				t.Errorf("GetRangeIndices(%s) returned %d indices, expected %d",
					tt.provider, len(indices), tt.expected)
			}

			// Check that all indices contain CREATE INDEX
			if tt.provider != GraphProviderKuzu {
				for _, index := range indices {
					if !strings.Contains(index, "CREATE INDEX") {
						t.Errorf("Index should contain 'CREATE INDEX': %s", index)
					}
				}
			}
		})
	}
}

func TestGetFulltextIndices(t *testing.T) {
	tests := []struct {
		provider GraphProvider
		expected int // expected number of indices
	}{
		{GraphProviderNeo4j, 4},     // Neo4j has 4 fulltext indices
		{GraphProviderFalkorDB, 4},  // FalkorDB has 4 fulltext indices
		{GraphProviderKuzu, 4},      // Kuzu has 4 fulltext indices
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			indices := GetFulltextIndices(tt.provider)
			if len(indices) != tt.expected {
				t.Errorf("GetFulltextIndices(%s) returned %d indices, expected %d",
					tt.provider, len(indices), tt.expected)
			}

			// Check that all indices are appropriate for the provider
			for _, index := range indices {
				switch tt.provider {
				case GraphProviderNeo4j:
					if !strings.Contains(index, "FULLTEXT INDEX") {
						t.Errorf("Neo4j index should contain 'FULLTEXT INDEX': %s", index)
					}
				case GraphProviderFalkorDB:
					if !strings.Contains(index, "FULLTEXT INDEX") {
						t.Errorf("FalkorDB index should contain 'FULLTEXT INDEX': %s", index)
					}
				case GraphProviderKuzu:
					if !strings.Contains(index, "CREATE_FTS_INDEX") {
						t.Errorf("Kuzu index should contain 'CREATE_FTS_INDEX': %s", index)
					}
				}
			}
		})
	}
}

func TestGetNodesQuery(t *testing.T) {
	tests := []struct {
		provider  GraphProvider
		indexName string
		query     string
		limit     int
		contains  string
	}{
		{GraphProviderNeo4j, "node_name_and_summary", "test", 10, "db.index.fulltext.queryNodes"},
		{GraphProviderFalkorDB, "node_name_and_summary", "test", 10, "db.idx.fulltext.queryNodes"},
		{GraphProviderKuzu, "node_name_and_summary", "test", 10, "QUERY_FTS_INDEX"},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			query := GetNodesQuery(tt.indexName, tt.query, tt.limit, tt.provider)
			if !strings.Contains(query, tt.contains) {
				t.Errorf("Query should contain '%s': %s", tt.contains, query)
			}
		})
	}
}

func TestGetVectorCosineFuncQuery(t *testing.T) {
	tests := []struct {
		provider GraphProvider
		vec1     string
		vec2     string
		contains string
	}{
		{GraphProviderNeo4j, "n.embedding", "m.embedding", "vector.similarity.cosine"},
		{GraphProviderFalkorDB, "n.embedding", "m.embedding", "vec.cosineDistance"},
		{GraphProviderKuzu, "n.embedding", "m.embedding", "array_cosine_similarity"},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			query := GetVectorCosineFuncQuery(tt.vec1, tt.vec2, tt.provider)
			if !strings.Contains(query, tt.contains) {
				t.Errorf("Query should contain '%s': %s", tt.contains, query)
			}
		})
	}
}

func TestQueryBuilder(t *testing.T) {
	builder := NewQueryBuilder(GraphProviderNeo4j)

	// Test provider getter
	if builder.GetProvider() != GraphProviderNeo4j {
		t.Errorf("Expected provider to be Neo4j, got %s", builder.GetProvider())
	}

	// Test provider setter
	builder.SetProvider(GraphProviderKuzu)
	if builder.GetProvider() != GraphProviderKuzu {
		t.Errorf("Expected provider to be Kuzu, got %s", builder.GetProvider())
	}

	// Test query building methods
	nodeQuery := builder.BuildFulltextNodeQuery("node_name_and_summary", "test", 10)
	if !strings.Contains(nodeQuery, "QUERY_FTS_INDEX") {
		t.Errorf("Kuzu node query should contain 'QUERY_FTS_INDEX': %s", nodeQuery)
	}

	relQuery := builder.BuildFulltextRelationshipQuery("edge_name_and_fact", 10)
	if !strings.Contains(relQuery, "QUERY_FTS_INDEX") {
		t.Errorf("Kuzu relationship query should contain 'QUERY_FTS_INDEX': %s", relQuery)
	}

	cosineQuery := builder.BuildCosineSimilarityQuery("n.embedding", "m.embedding")
	if !strings.Contains(cosineQuery, "array_cosine_similarity") {
		t.Errorf("Kuzu cosine query should contain 'array_cosine_similarity': %s", cosineQuery)
	}

	// Test index queries
	rangeIndices := builder.GetRangeIndexQueries()
	if len(rangeIndices) != 0 { // Kuzu should have 0 range indices
		t.Errorf("Kuzu should have 0 range indices, got %d", len(rangeIndices))
	}

	fulltextIndices := builder.GetFulltextIndexQueries()
	if len(fulltextIndices) != 4 { // Should have 4 fulltext indices
		t.Errorf("Should have 4 fulltext indices, got %d", len(fulltextIndices))
	}
}

func TestEscapeQueryString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with spaces", "with spaces"},
		{`with "quotes"`, `with \"quotes\"`},
		{"with + and -", `with \+ and \-`},
		{"with (parens)", `with \(parens\)`},
		{"with [brackets]", `with \[brackets\]`},
		{"with {braces}", `with \{braces\}`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := EscapeQueryString(tt.input)
			if result != tt.expected {
				t.Errorf("EscapeQueryString(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuildParameterizedQuery(t *testing.T) {
	query := "MATCH (n) WHERE n.id = $id RETURN n"
	params := map[string]interface{}{
		"id":        "test-id",
		"database_": "neo4j", // Should be filtered out
		"routing_":  "write", // Should be filtered out
		"valid":     "value",
		"nil_value": nil, // Should be filtered out
	}

	resultQuery, resultParams := BuildParameterizedQuery(query, params)

	// Query should remain unchanged
	if resultQuery != query {
		t.Errorf("Query should remain unchanged")
	}

	// Should only contain valid parameters
	expectedParams := map[string]interface{}{
		"id":    "test-id",
		"valid": "value",
	}

	if len(resultParams) != len(expectedParams) {
		t.Errorf("Expected %d parameters, got %d", len(expectedParams), len(resultParams))
	}

	for key, value := range expectedParams {
		if resultParams[key] != value {
			t.Errorf("Expected param %s = %v, got %v", key, value, resultParams[key])
		}
	}
}