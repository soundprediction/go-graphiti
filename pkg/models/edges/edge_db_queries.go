package edges

import (
	"github.com/soundprediction/go-graphiti/pkg/driver"
)

// EPISODIC_EDGE_SAVE query constant
const EPISODIC_EDGE_SAVE = `
    MATCH (episode:Episodic {uuid: $episode_uuid})
    MATCH (node:Entity {uuid: $entity_uuid})
    MERGE (episode)-[e:MENTIONS {uuid: $uuid}]->(node)
    SET
        e.group_id = $group_id,
        e.created_at = $created_at
    RETURN e.uuid AS uuid
`

// GetEpisodicEdgeSaveBulkQuery returns the bulk save query for episodic edges based on provider
func GetEpisodicEdgeSaveBulkQuery(provider driver.GraphProvider) string {
	if provider == driver.GraphProviderKuzu {
		return `
            MATCH (episode:Episodic {uuid: $source_node_uuid})
            MATCH (node:Entity {uuid: $target_node_uuid})
            MERGE (episode)-[e:MENTIONS {uuid: $uuid}]->(node)
            SET
                e.group_id = $group_id,
                e.created_at = $created_at
            RETURN e.uuid AS uuid
        `
	}

	return `
        UNWIND $episodic_edges AS edge
        MATCH (episode:Episodic {uuid: edge.source_node_uuid})
        MATCH (node:Entity {uuid: edge.target_node_uuid})
        MERGE (episode)-[e:MENTIONS {uuid: edge.uuid}]->(node)
        SET
            e.group_id = edge.group_id,
            e.created_at = edge.created_at
        RETURN e.uuid AS uuid
    `
}

// EPISODIC_EDGE_RETURN query constant
const EPISODIC_EDGE_RETURN = `
    e.uuid AS uuid,
    e.group_id AS group_id,
    n.uuid AS source_node_uuid,
    m.uuid AS target_node_uuid,
    e.created_at AS created_at
`

// GetEntityEdgeSaveQuery returns the entity edge save query based on provider and AOSS configuration
func GetEntityEdgeSaveQuery(provider driver.GraphProvider, hasAOSS bool) string {
	switch provider {
	case driver.GraphProviderFalkorDB:
		return `
                MATCH (source:Entity {uuid: $edge_data.source_uuid})
                MATCH (target:Entity {uuid: $edge_data.target_uuid})
                MERGE (source)-[e:RELATES_TO {uuid: $edge_data.uuid}]->(target)
                SET e = $edge_data
                RETURN e.uuid AS uuid
            `
	case driver.GraphProviderNeptune:
		return `
                MATCH (source:Entity {uuid: $edge_data.source_uuid})
                MATCH (target:Entity {uuid: $edge_data.target_uuid})
                MERGE (source)-[e:RELATES_TO {uuid: $edge_data.uuid}]->(target)
                SET e = removeKeyFromMap(removeKeyFromMap($edge_data, "fact_embedding"), "episodes")
                SET e.fact_embedding = join([x IN coalesce($edge_data.fact_embedding, []) | toString(x) ], ",")
                SET e.episodes = join($edge_data.episodes, ",")
                RETURN $edge_data.uuid AS uuid
            `
	case driver.GraphProviderKuzu:
		return `
                MATCH (source:Entity {uuid: $source_uuid})
                MATCH (target:Entity {uuid: $target_uuid})
                MERGE (source)-[:RELATES_TO]->(e:RelatesToNode_ {uuid: $uuid})-[:RELATES_TO]->(target)
                SET
                    e.group_id = $group_id,
                    e.created_at = $created_at,
                    e.name = $name,
                    e.fact = $fact,
                    e.fact_embedding = $fact_embedding,
                    e.episodes = $episodes,
                    e.expired_at = $expired_at,
                    e.valid_at = $valid_at,
                    e.invalid_at = $invalid_at,
                    e.attributes = $attributes
                RETURN e.uuid AS uuid
            `
	default: // Neo4j
		saveEmbeddingQuery := ""
		if !hasAOSS {
			saveEmbeddingQuery = `WITH e CALL db.create.setRelationshipVectorProperty(e, "fact_embedding", $edge_data.fact_embedding)`
		}

		return `
                        MATCH (source:Entity {uuid: $edge_data.source_uuid})
                        MATCH (target:Entity {uuid: $edge_data.target_uuid})
                        MERGE (source)-[e:RELATES_TO {uuid: $edge_data.uuid}]->(target)
                        SET e = $edge_data
                        ` + saveEmbeddingQuery + `
                RETURN e.uuid AS uuid
                `
	}
}

// GetEntityEdgeSaveBulkQuery returns the bulk entity edge save query based on provider and AOSS configuration
func GetEntityEdgeSaveBulkQuery(provider driver.GraphProvider, hasAOSS bool) string {
	switch provider {
	case driver.GraphProviderFalkorDB:
		return `
                UNWIND $entity_edges AS edge
                MATCH (source:Entity {uuid: edge.source_node_uuid})
                MATCH (target:Entity {uuid: edge.target_node_uuid})
                MERGE (source)-[r:RELATES_TO {uuid: edge.uuid}]->(target)
                SET r = {uuid: edge.uuid, name: edge.name, group_id: edge.group_id, fact: edge.fact, episodes: edge.episodes,
                created_at: edge.created_at, expired_at: edge.expired_at, valid_at: edge.valid_at, invalid_at: edge.invalid_at, fact_embedding: vecf32(edge.fact_embedding)}
                WITH r, edge
                RETURN edge.uuid AS uuid
            `
	case driver.GraphProviderNeptune:
		return `
                UNWIND $entity_edges AS edge
                MATCH (source:Entity {uuid: edge.source_node_uuid})
                MATCH (target:Entity {uuid: edge.target_node_uuid})
                MERGE (source)-[r:RELATES_TO {uuid: edge.uuid}]->(target)
                SET r = removeKeyFromMap(removeKeyFromMap(edge, "fact_embedding"), "episodes")
                SET r.fact_embedding = join([x IN coalesce(edge.fact_embedding, []) | toString(x) ], ",")
                SET r.episodes = join(edge.episodes, ",")
                RETURN edge.uuid AS uuid
            `
	case driver.GraphProviderKuzu:
		return `
                MATCH (source:Entity {uuid: $source_node_uuid})
                MATCH (target:Entity {uuid: $target_node_uuid})
                MERGE (source)-[:RELATES_TO]->(e:RelatesToNode_ {uuid: $uuid})-[:RELATES_TO]->(target)
                SET
                    e.group_id = $group_id,
                    e.created_at = $created_at,
                    e.name = $name,
                    e.fact = $fact,
                    e.fact_embedding = $fact_embedding,
                    e.episodes = $episodes,
                    e.expired_at = $expired_at,
                    e.valid_at = $valid_at,
                    e.invalid_at = $invalid_at,
                    e.attributes = $attributes
                RETURN e.uuid AS uuid
            `
	default: // Neo4j
		saveEmbeddingQuery := ""
		if !hasAOSS {
			saveEmbeddingQuery = `WITH e, edge CALL db.create.setRelationshipVectorProperty(e, "fact_embedding", edge.fact_embedding)`
		}

		return `
                    UNWIND $entity_edges AS edge
                    MATCH (source:Entity {uuid: edge.source_node_uuid})
                    MATCH (target:Entity {uuid: edge.target_node_uuid})
                    MERGE (source)-[e:RELATES_TO {uuid: edge.uuid}]->(target)
                    SET e = edge
                    ` + saveEmbeddingQuery + `
                    RETURN edge.uuid AS uuid
            `
	}
}

// GetEntityEdgeReturnQuery returns the entity edge return query based on provider
// Note: fact_embedding is not returned by default and must be manually loaded using load_fact_embedding().
func GetEntityEdgeReturnQuery(provider driver.GraphProvider) string {
	if provider == driver.GraphProviderNeptune {
		return `
        e.uuid AS uuid,
        n.uuid AS source_node_uuid,
        m.uuid AS target_node_uuid,
        e.group_id AS group_id,
        e.name AS name,
        e.fact AS fact,
        split(e.episodes, ',') AS episodes,
        e.created_at AS created_at,
        e.expired_at AS expired_at,
        e.valid_at AS valid_at,
        e.invalid_at AS invalid_at,
        properties(e) AS attributes
    `
	}

	attributesClause := "properties(e) AS attributes"
	if provider == driver.GraphProviderKuzu {
		attributesClause = "e.attributes AS attributes"
	}

	return `
        e.uuid AS uuid,
        n.uuid AS source_node_uuid,
        m.uuid AS target_node_uuid,
        e.group_id AS group_id,
        e.created_at AS created_at,
        e.name AS name,
        e.fact AS fact,
        e.episodes AS episodes,
        e.expired_at AS expired_at,
        e.valid_at AS valid_at,
        e.invalid_at AS invalid_at,
        ` + attributesClause
}

// GetCommunityEdgeSaveQuery returns the community edge save query based on provider
func GetCommunityEdgeSaveQuery(provider driver.GraphProvider) string {
	switch provider {
	case driver.GraphProviderFalkorDB:
		return `
                MATCH (community:Community {uuid: $community_uuid})
                MATCH (node {uuid: $entity_uuid})
                MERGE (community)-[e:HAS_MEMBER {uuid: $uuid}]->(node)
                SET e = {uuid: $uuid, group_id: $group_id, created_at: $created_at}
                RETURN e.uuid AS uuid
            `
	case driver.GraphProviderNeptune:
		return `
                MATCH (community:Community {uuid: $community_uuid})
                MATCH (node {uuid: $entity_uuid})
                WHERE node:Entity OR node:Community
                MERGE (community)-[r:HAS_MEMBER {uuid: $uuid}]->(node)
                SET r.uuid= $uuid
                SET r.group_id= $group_id
                SET r.created_at= $created_at
                RETURN r.uuid AS uuid
            `
	case driver.GraphProviderKuzu:
		return `
                MATCH (community:Community {uuid: $community_uuid})
                MATCH (node:Entity {uuid: $entity_uuid})
                MERGE (community)-[e:HAS_MEMBER {uuid: $uuid}]->(node)
                SET
                    e.group_id = $group_id,
                    e.created_at = $created_at
                RETURN e.uuid AS uuid
                UNION
                MATCH (community:Community {uuid: $community_uuid})
                MATCH (node:Community {uuid: $entity_uuid})
                MERGE (community)-[e:HAS_MEMBER {uuid: $uuid}]->(node)
                SET
                    e.group_id = $group_id,
                    e.created_at = $created_at
                RETURN e.uuid AS uuid
            `
	default: // Neo4j
		return `
                MATCH (community:Community {uuid: $community_uuid})
                MATCH (node:Entity | Community {uuid: $entity_uuid})
                MERGE (community)-[e:HAS_MEMBER {uuid: $uuid}]->(node)
                SET e = {uuid: $uuid, group_id: $group_id, created_at: $created_at}
                RETURN e.uuid AS uuid
            `
	}
}

// COMMUNITY_EDGE_RETURN query constant
const COMMUNITY_EDGE_RETURN = `
    e.uuid AS uuid,
    e.group_id AS group_id,
    n.uuid AS source_node_uuid,
    m.uuid AS target_node_uuid,
    e.created_at AS created_at
`