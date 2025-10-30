package types

// Neighbor represents a neighboring node with edge count
type Neighbor struct {
	NodeUUID  string `json:"node_uuid"`
	EdgeCount int    `json:"edge_count"`
}
