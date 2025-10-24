package search

// CombinedHybridSearchRRF performs a hybrid search with RRF reranking over edges, nodes, and communities
var CombinedHybridSearchRRF = &SearchConfig{
	EdgeConfig: &EdgeSearchConfig{
		SearchMethods: []SearchMethod{BM25, CosineSimilarity},
		Reranker:      RRFRerankType,
	},
	NodeConfig: &NodeSearchConfig{
		SearchMethods: []SearchMethod{BM25, CosineSimilarity},
		Reranker:      RRFRerankType,
	},
	EpisodeConfig: &EpisodeSearchConfig{
		SearchMethods: []SearchMethod{BM25},
		Reranker:      RRFRerankType,
	},
	CommunityConfig: &CommunitySearchConfig{
		SearchMethods: []SearchMethod{BM25, CosineSimilarity},
		Reranker:      RRFRerankType,
	},
}

// CombinedHybridSearchMMR performs a hybrid search with MMR reranking over edges, nodes, and communities
var CombinedHybridSearchMMR = &SearchConfig{
	EdgeConfig: &EdgeSearchConfig{
		SearchMethods: []SearchMethod{BM25, CosineSimilarity},
		Reranker:      MMRRerankType,
		MMRLambda:     1.0,
	},
	NodeConfig: &NodeSearchConfig{
		SearchMethods: []SearchMethod{BM25, CosineSimilarity},
		Reranker:      MMRRerankType,
		MMRLambda:     1.0,
	},
	EpisodeConfig: &EpisodeSearchConfig{
		SearchMethods: []SearchMethod{BM25},
		Reranker:      RRFRerankType,
	},
	CommunityConfig: &CommunitySearchConfig{
		SearchMethods: []SearchMethod{BM25, CosineSimilarity},
		Reranker:      MMRRerankType,
		MMRLambda:     1.0,
	},
}

// CombinedHybridSearchCrossEncoder performs a full-text search, similarity search, and BFS with cross_encoder reranking
var CombinedHybridSearchCrossEncoder = &SearchConfig{
	EdgeConfig: &EdgeSearchConfig{
		SearchMethods: []SearchMethod{BM25, CosineSimilarity, BreadthFirstSearch},
		Reranker:      CrossEncoderRerankType,
	},
	NodeConfig: &NodeSearchConfig{
		SearchMethods: []SearchMethod{BM25, CosineSimilarity, BreadthFirstSearch},
		Reranker:      CrossEncoderRerankType,
	},
	EpisodeConfig: &EpisodeSearchConfig{
		SearchMethods: []SearchMethod{BM25},
		Reranker:      CrossEncoderRerankType,
	},
	CommunityConfig: &CommunitySearchConfig{
		SearchMethods: []SearchMethod{BM25, CosineSimilarity},
		Reranker:      CrossEncoderRerankType,
	},
}

// EdgeHybridSearchRRF performs a hybrid search over edges with RRF reranking
var EdgeHybridSearchRRF = &SearchConfig{
	EdgeConfig: &EdgeSearchConfig{
		SearchMethods: []SearchMethod{BM25, CosineSimilarity},
		Reranker:      RRFRerankType,
	},
}

// EdgeHybridSearchMMR performs a hybrid search over edges with MMR reranking
var EdgeHybridSearchMMR = &SearchConfig{
	EdgeConfig: &EdgeSearchConfig{
		SearchMethods: []SearchMethod{BM25, CosineSimilarity},
		Reranker:      MMRRerankType,
	},
}

// EdgeHybridSearchNodeDistance performs a hybrid search over edges with node distance reranking
var EdgeHybridSearchNodeDistance = &SearchConfig{
	EdgeConfig: &EdgeSearchConfig{
		SearchMethods: []SearchMethod{BM25, CosineSimilarity},
		Reranker:      NodeDistanceRerankType,
	},
}

// EdgeHybridSearchEpisodeMentions performs a hybrid search over edges with episode mention reranking
var EdgeHybridSearchEpisodeMentions = &SearchConfig{
	EdgeConfig: &EdgeSearchConfig{
		SearchMethods: []SearchMethod{BM25, CosineSimilarity},
		Reranker:      EpisodeMentionsRerankType,
	},
}

// EdgeHybridSearchCrossEncoder performs a hybrid search over edges with cross encoder reranking
var EdgeHybridSearchCrossEncoder = &SearchConfig{
	EdgeConfig: &EdgeSearchConfig{
		SearchMethods: []SearchMethod{BM25, CosineSimilarity, BreadthFirstSearch},
		Reranker:      CrossEncoderRerankType,
	},
	Limit: 10,
}

// NodeHybridSearchRRF performs a hybrid search over nodes with RRF reranking
var NodeHybridSearchRRF = &SearchConfig{
	NodeConfig: &NodeSearchConfig{
		SearchMethods: []SearchMethod{BM25, CosineSimilarity},
		Reranker:      RRFRerankType,
	},
}

// NodeHybridSearchMMR performs a hybrid search over nodes with MMR reranking
var NodeHybridSearchMMR = &SearchConfig{
	NodeConfig: &NodeSearchConfig{
		SearchMethods: []SearchMethod{BM25, CosineSimilarity},
		Reranker:      MMRRerankType,
	},
}

// NodeHybridSearchNodeDistance performs a hybrid search over nodes with node distance reranking
var NodeHybridSearchNodeDistance = &SearchConfig{
	NodeConfig: &NodeSearchConfig{
		SearchMethods: []SearchMethod{BM25, CosineSimilarity},
		Reranker:      NodeDistanceRerankType,
	},
}

// NodeHybridSearchEpisodeMentions performs a hybrid search over nodes with episode mentions reranking
var NodeHybridSearchEpisodeMentions = &SearchConfig{
	NodeConfig: &NodeSearchConfig{
		SearchMethods: []SearchMethod{BM25, CosineSimilarity},
		Reranker:      EpisodeMentionsRerankType,
	},
}

// NodeHybridSearchCrossEncoder performs a hybrid search over nodes with cross encoder reranking
var NodeHybridSearchCrossEncoder = &SearchConfig{
	NodeConfig: &NodeSearchConfig{
		SearchMethods: []SearchMethod{BM25, CosineSimilarity, BreadthFirstSearch},
		Reranker:      CrossEncoderRerankType,
	},
	Limit: 10,
}

// CommunityHybridSearchRRF performs a hybrid search over communities with RRF reranking
var CommunityHybridSearchRRF = &SearchConfig{
	CommunityConfig: &CommunitySearchConfig{
		SearchMethods: []SearchMethod{BM25, CosineSimilarity},
		Reranker:      RRFRerankType,
	},
}

// CommunityHybridSearchMMR performs a hybrid search over communities with MMR reranking
var CommunityHybridSearchMMR = &SearchConfig{
	CommunityConfig: &CommunitySearchConfig{
		SearchMethods: []SearchMethod{BM25, CosineSimilarity},
		Reranker:      MMRRerankType,
	},
}

// CommunityHybridSearchCrossEncoder performs a hybrid search over communities with cross encoder reranking
var CommunityHybridSearchCrossEncoder = &SearchConfig{
	CommunityConfig: &CommunitySearchConfig{
		SearchMethods: []SearchMethod{BM25, CosineSimilarity},
		Reranker:      CrossEncoderRerankType,
	},
	Limit: 3,
}
