package entity

// SearchQuery represents a semantic search request
type SearchQuery struct {
	Query        string  `json:"query" validate:"required,min=1"`
	Namespace    *string `json:"namespace,omitempty"`
	MinScore     *int    `json:"min_score,omitempty" validate:"omitempty,min=0,max=100"`
	VerifiedOnly bool    `json:"verified_only,omitempty"`
	Limit        int     `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
	Offset       int     `json:"offset,omitempty" validate:"omitempty,min=0"`
}

// SearchResult represents a single search result with similarity score
type SearchResult struct {
	Server     Server   `json:"server"`
	Score      float64  `json:"score"`
	Highlights []string `json:"highlights"`
}
