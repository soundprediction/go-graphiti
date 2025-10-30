package community

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/soundprediction/go-graphiti/pkg/driver"
	"github.com/soundprediction/go-graphiti/pkg/embedder"
	"github.com/soundprediction/go-graphiti/pkg/llm"
	"github.com/soundprediction/go-graphiti/pkg/types"
)

const (
	// MaxCommunityBuildConcurrency limits concurrent community building operations
	MaxCommunityBuildConcurrency = 10
)

// Builder provides community building operations for knowledge graphs
type Builder struct {
	driver   driver.GraphDriver
	llm      llm.Client
	embedder embedder.Client
}

// NewBuilder creates a new community builder
func NewBuilder(driver driver.GraphDriver, llmClient llm.Client, embedderClient embedder.Client) *Builder {
	return &Builder{
		driver:   driver,
		llm:      llmClient,
		embedder: embedderClient,
	}
}

// BuildCommunitiesResult represents the result of community building
type BuildCommunitiesResult struct {
	CommunityNodes []*types.Node `json:"community_nodes"`
	CommunityEdges []*types.Edge `json:"community_edges"`
}

// GetCommunityClusters detects community clusters using label propagation algorithm
func (b *Builder) GetCommunityClusters(ctx context.Context, groupIDs []string) ([][]*types.Node, error) {
	if len(groupIDs) == 0 {
		// Get all group IDs if none specified
		allGroupIDs, err := b.getAllGroupIDs(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get group IDs: %w", err)
		}
		groupIDs = allGroupIDs
	}
	var allClusters [][]*types.Node

	for _, groupID := range groupIDs {
		// Get all entity nodes for this group
		nodes, err := b.getEntityNodesByGroup(ctx, groupID)
		if err != nil {
			return nil, fmt.Errorf("failed to get entity nodes for group %s: %w", groupID, err)
		}

		// Build adjacency projection
		projection, err := b.buildProjection(ctx, nodes, groupID)
		if err != nil {
			return nil, fmt.Errorf("failed to build projection for group %s: %w", groupID, err)
		}

		// Apply label propagation algorithm
		clusterUUIDs := b.labelPropagation(projection)

		// Convert UUID clusters to node clusters
		for _, cluster := range clusterUUIDs {
			clusterNodes, err := b.getNodesByUUIDs(ctx, cluster, groupID)
			if err != nil {
				return nil, fmt.Errorf("failed to get nodes for cluster: %w", err)
			}
			if len(clusterNodes) > 0 {
				allClusters = append(allClusters, clusterNodes)
			}
		}
	}

	return allClusters, nil
}

// BuildCommunities builds communities from entity clusters
func (b *Builder) BuildCommunities(ctx context.Context, groupIDs []string, logger *slog.Logger) (*BuildCommunitiesResult, error) {
	// Get community clusters
	clusters, err := b.GetCommunityClusters(ctx, groupIDs)
	if logger != nil {
		logger.Info("Clustering", "num_clusters", len(clusters), "num_groups", len(groupIDs))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get community clusters: %w", err)
	}

	// Limit concurrency
	semaphore := make(chan struct{}, MaxCommunityBuildConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	var allCommunityNodes []*types.Node
	var allCommunityEdges []*types.Edge
	var buildErrors []error

	// Build communities concurrently
	for _, cluster := range clusters {
		wg.Add(1)
		go func(cluster []*types.Node) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			communityNode, communityEdges, err := b.buildCommunity(ctx, cluster)
			if err != nil {
				mu.Lock()
				buildErrors = append(buildErrors, err)
				mu.Unlock()
				return
			}

			mu.Lock()
			allCommunityNodes = append(allCommunityNodes, communityNode)
			allCommunityEdges = append(allCommunityEdges, communityEdges...)
			mu.Unlock()
		}(cluster)
	}

	wg.Wait()

	if len(buildErrors) > 0 {
		return nil, fmt.Errorf("failed to build communities: %v", buildErrors[0])
	}

	return &BuildCommunitiesResult{
		CommunityNodes: allCommunityNodes,
		CommunityEdges: allCommunityEdges,
	}, nil
}

// buildCommunity builds a single community from a cluster of entities
func (b *Builder) buildCommunity(ctx context.Context, cluster []*types.Node) (*types.Node, []*types.Edge, error) {
	if len(cluster) == 0 {
		return nil, nil, fmt.Errorf("empty cluster")
	}

	// Extract summaries for hierarchical summarization
	summaries := make([]string, len(cluster))
	for i, node := range cluster {
		summaries[i] = node.Summary
	}

	// Hierarchical summarization
	finalSummary, err := b.hierarchicalSummarize(ctx, summaries)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to summarize cluster: %w", err)
	}

	// Generate community name
	communityName, err := b.generateCommunityName(ctx, finalSummary)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate community name: %w", err)
	}

	// Create community node
	now := time.Now().UTC()
	communityNode := &types.Node{
		ID:        generateUUID(),
		Name:      communityName,
		Type:      types.CommunityNodeType,
		GroupID:   cluster[0].GroupID,
		CreatedAt: now,
		UpdatedAt: now,
		Summary:   finalSummary,
		ValidFrom: now,
		Metadata:  make(map[string]interface{}),
	}

	// Generate embedding for community name
	if err := b.generateCommunityEmbedding(ctx, communityNode); err != nil {
		return nil, nil, fmt.Errorf("failed to generate community embedding: %w", err)
	}

	// Build community edges (HAS_MEMBER relationships)
	communityEdges := b.buildCommunityEdges(cluster, communityNode, now)

	return communityNode, communityEdges, nil
}

// hierarchicalSummarize performs hierarchical summarization of node summaries
func (b *Builder) hierarchicalSummarize(ctx context.Context, summaries []string) (string, error) {
	if len(summaries) == 0 {
		return "", fmt.Errorf("no summaries to process")
	}

	if len(summaries) == 1 {
		return summaries[0], nil
	}

	currentSummaries := make([]string, len(summaries))
	copy(currentSummaries, summaries)

	for len(currentSummaries) > 1 {
		var newSummaries []string
		var oddOneOut string

		// Handle odd number of summaries
		if len(currentSummaries)%2 == 1 {
			oddOneOut = currentSummaries[len(currentSummaries)-1]
			currentSummaries = currentSummaries[:len(currentSummaries)-1]
		}

		// Process pairs concurrently
		pairCount := len(currentSummaries) / 2
		results := make([]string, pairCount)

		var wg sync.WaitGroup
		var mu sync.Mutex
		var summarizeErrors []error

		for i := 0; i < pairCount; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				left := currentSummaries[idx]
				right := currentSummaries[idx+pairCount]

				pairSummary, err := b.summarizePair(ctx, left, right)
				if err != nil {
					mu.Lock()
					summarizeErrors = append(summarizeErrors, err)
					mu.Unlock()
					return
				}

				mu.Lock()
				results[idx] = pairSummary
				mu.Unlock()
			}(i)
		}

		wg.Wait()

		if len(summarizeErrors) > 0 {
			return "", fmt.Errorf("failed to summarize pairs: %w", summarizeErrors[0])
		}

		newSummaries = results
		if oddOneOut != "" {
			newSummaries = append(newSummaries, oddOneOut)
		}

		currentSummaries = newSummaries
	}

	return currentSummaries[0], nil
}

// summarizePair summarizes two text summaries into one
func (b *Builder) summarizePair(ctx context.Context, left, right string) (string, error) {
	messages := []llm.Message{
		{
			Role:    llm.RoleSystem,
			Content: `You are an expert at synthesizing information. Given two entity summaries, create a single comprehensive summary that captures the key information from both. The summary should be concise (under 250 words) and maintain the most important details.`,
		},
		{
			Role: llm.RoleUser,
			Content: fmt.Sprintf(`Please summarize these two entity summaries into one comprehensive summary:

Summary 1: %s

Summary 2: %s

Provide a single summary that captures the essential information from both:`, left, right),
		},
	}

	response, err := b.llm.Chat(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to get LLM response for pair summarization: %w", err)
	}

	return response.Content, nil
}

// generateCommunityName generates a descriptive name for a community based on its summary
func (b *Builder) generateCommunityName(ctx context.Context, summary string) (string, error) {
	messages := []llm.Message{
		{
			Role:    llm.RoleSystem,
			Content: `You are an expert at creating concise, descriptive names. Given a summary, create a brief descriptive name (1-5 words) that captures the essence of the content.`,
		},
		{
			Role: llm.RoleUser,
			Content: fmt.Sprintf(`Based on this summary, provide a brief descriptive name (1-5 words):

%s

Name:`, summary),
		},
	}

	response, err := b.llm.Chat(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to generate community name: %w", err)
	}

	return response.Content, nil
}

// generateCommunityEmbedding generates an embedding for the community name
func (b *Builder) generateCommunityEmbedding(ctx context.Context, community *types.Node) error {
	embedding, err := b.embedder.EmbedSingle(ctx, community.Name)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	community.Embedding = embedding
	return nil
}

// buildCommunityEdges creates HAS_MEMBER edges between community and entity nodes
func (b *Builder) buildCommunityEdges(entityNodes []*types.Node, communityNode *types.Node, createdAt time.Time) []*types.Edge {
	edges := make([]*types.Edge, len(entityNodes))

	for i, entityNode := range entityNodes {
		edge := types.NewEntityEdge(
			generateUUID(),
			communityNode.ID,
			entityNode.ID,
			communityNode.GroupID,
			"HAS_MEMBER",
			types.CommunityEdgeType,
		)
		edge.UpdatedAt = createdAt
		edge.ValidFrom = createdAt
		edge.SourceIDs = []string{communityNode.ID}
		edge.Metadata = make(map[string]interface{})
		edges[i] = edge
	}

	return edges
}

// RemoveCommunities removes all community nodes and edges from the graph
func (b *Builder) RemoveCommunities(ctx context.Context) error {
	query := "MATCH (c:Community) DETACH DELETE c"

	// Execute based on driver type
	if kuzuDriver, ok := b.driver.(*driver.KuzuDriver); ok {
		// For Kuzu, use the executeQuery method
		_, _, _, err := kuzuDriver.ExecuteQuery(query, nil)
		return err
	}

	// For other drivers, implement accordingly
	return fmt.Errorf("driver type not supported for community removal")
}

// generateUUID generates a simple UUID-like string
func generateUUID() string {
	return fmt.Sprintf("comm_%d", time.Now().UnixNano())
}
