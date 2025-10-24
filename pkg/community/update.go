package community

import (
	"context"
	"fmt"
	"time"

	"github.com/soundprediction/go-graphiti/pkg/driver"
	"github.com/soundprediction/go-graphiti/pkg/types"
)

// DetermineEntityCommunityResult represents the result of determining an entity's community
type DetermineEntityCommunityResult struct {
	Community *types.Node
	IsNew     bool
}

// UpdateCommunityResult represents the result of updating a community
type UpdateCommunityResult struct {
	CommunityNodes []*types.Node
	CommunityEdges []*types.Edge
}

// DetermineEntityCommunity determines which community an entity belongs to
func (b *Builder) DetermineEntityCommunity(ctx context.Context, entity *types.Node) (*DetermineEntityCommunityResult, error) {
	// First check if the entity is already part of a community
	existingCommunity, err := b.getExistingCommunity(ctx, entity.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing community: %w", err)
	}

	if existingCommunity != nil {
		return &DetermineEntityCommunityResult{
			Community: existingCommunity,
			IsNew:     false,
		}, nil
	}

	// Find the most common community among connected entities
	modalCommunity, err := b.findModalCommunity(ctx, entity.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to find modal community: %w", err)
	}

	if modalCommunity == nil {
		return &DetermineEntityCommunityResult{
			Community: nil,
			IsNew:     false,
		}, nil
	}

	return &DetermineEntityCommunityResult{
		Community: modalCommunity,
		IsNew:     true,
	}, nil
}

// UpdateCommunity updates a community when a new entity is added
func (b *Builder) UpdateCommunity(ctx context.Context, entity *types.Node) (*UpdateCommunityResult, error) {
	// Determine which community the entity should belong to
	result, err := b.DetermineEntityCommunity(ctx, entity)
	if err != nil {
		return nil, fmt.Errorf("failed to determine entity community: %w", err)
	}

	if result.Community == nil {
		return &UpdateCommunityResult{
			CommunityNodes: []*types.Node{},
			CommunityEdges: []*types.Edge{},
		}, nil
	}

	community := result.Community

	// Create new summary by combining entity and community summaries
	newSummary, err := b.summarizePair(ctx, entity.Summary, community.Summary)
	if err != nil {
		return nil, fmt.Errorf("failed to create new summary: %w", err)
	}

	// Generate new name based on the updated summary
	newName, err := b.generateCommunityName(ctx, newSummary)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new community name: %w", err)
	}

	// Update community
	community.Summary = newSummary
	community.Name = newName
	community.UpdatedAt = time.Now().UTC()

	// Generate new embedding for the updated name
	if err := b.generateCommunityEmbedding(ctx, community); err != nil {
		return nil, fmt.Errorf("failed to generate community embedding: %w", err)
	}

	// Save updated community
	if err := b.driver.UpsertNode(ctx, community); err != nil {
		return nil, fmt.Errorf("failed to save updated community: %w", err)
	}

	var communityEdges []*types.Edge

	// If this is a new membership, create HAS_MEMBER edge
	if result.IsNew {
		edge := types.NewEntityEdge(
			generateUUID(),
			community.ID,
			entity.ID,
			community.GroupID,
			"HAS_MEMBER",
			types.CommunityEdgeType,
		)
		edge.UpdatedAt = time.Now().UTC()
		edge.ValidFrom = time.Now().UTC()
		edge.SourceIDs = []string{community.ID}
		edge.Metadata = make(map[string]interface{})

		if err := b.driver.UpsertEdge(ctx, edge); err != nil {
			return nil, fmt.Errorf("failed to save community edge: %w", err)
		}

		communityEdges = append(communityEdges, edge)
	}

	return &UpdateCommunityResult{
		CommunityNodes: []*types.Node{community},
		CommunityEdges: communityEdges,
	}, nil
}

// getExistingCommunity checks if an entity is already part of a community
func (b *Builder) getExistingCommunity(ctx context.Context, entityUUID string) (*types.Node, error) {
	if kuzuDriver, ok := b.driver.(*driver.KuzuDriver); ok {
		return b.getExistingCommunityKuzu(ctx, kuzuDriver, entityUUID)
	}

	return nil, fmt.Errorf("non-Kuzu drivers not yet supported for getting existing community")
}

// getExistingCommunityKuzu gets existing community for Kuzu database
func (b *Builder) getExistingCommunityKuzu(ctx context.Context, kuzuDriver *driver.KuzuDriver, entityUUID string) (*types.Node, error) {
	// This would need proper implementation based on your Kuzu driver
	// For now, returning nil to indicate no existing community
	return nil, nil
}

// findModalCommunity finds the most common community among connected entities
func (b *Builder) findModalCommunity(ctx context.Context, entityUUID string) (*types.Node, error) {
	if kuzuDriver, ok := b.driver.(*driver.KuzuDriver); ok {
		return b.findModalCommunityKuzu(ctx, kuzuDriver, entityUUID)
	}

	return nil, fmt.Errorf("non-Kuzu drivers not yet supported for finding modal community")
}

// findModalCommunityKuzu finds modal community for Kuzu database
func (b *Builder) findModalCommunityKuzu(ctx context.Context, kuzuDriver interface{}, entityUUID string) (*types.Node, error) {
	// This would need proper implementation based on your Kuzu driver
	// For now, returning nil to indicate no modal community found
	return nil, nil
}
