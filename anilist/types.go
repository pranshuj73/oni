package anilist

// Anime represents an anime from AniList
type Anime struct {
	ID            int    `json:"id"`
	Title         Title  `json:"title"`
	CoverImage    Cover  `json:"coverImage"`
	StartDate     Date   `json:"startDate"`
	Episodes      *int   `json:"episodes"`
	Status        string `json:"status"`
	Description   string `json:"description"`
	AverageScore  *int   `json:"averageScore"`
	IsAdult       bool   `json:"isAdult"`
}

// Title represents anime titles
type Title struct {
	UserPreferred string `json:"userPreferred"`
	Romaji        string `json:"romaji"`
	English       string `json:"english"`
	Native        string `json:"native"`
}

// Cover represents cover image URLs
type Cover struct {
	ExtraLarge string `json:"extraLarge"`
	Large      string `json:"large"`
	Medium     string `json:"medium"`
}

// Date represents a date
type Date struct {
	Year  *int `json:"year"`
	Month *int `json:"month"`
	Day   *int `json:"day"`
}

// MediaListEntry represents a user's anime list entry
type MediaListEntry struct {
	ID        int    `json:"id"`
	MediaID   int    `json:"mediaId"`
	Status    string `json:"status"`
	Score     *float64 `json:"score"`
	Progress  int    `json:"progress"`
	Media     Anime  `json:"media"`
}

// MediaList represents a collection of list entries
type MediaList struct {
	Entries []MediaListEntry `json:"entries"`
}

// MediaListCollection represents the user's complete list
type MediaListCollection struct {
	Lists []MediaList `json:"lists"`
}

// SearchResponse represents search results
type SearchResponse struct {
	Page struct {
		Media []Anime `json:"media"`
	} `json:"Page"`
}

// ListResponse represents list query results
type ListResponse struct {
	MediaListCollection MediaListCollection `json:"MediaListCollection"`
}

// UserResponse represents user data
type UserResponse struct {
	Viewer struct {
		ID int `json:"id"`
	} `json:"Viewer"`
}

// UpdateResponse represents mutation response
type UpdateResponse struct {
	SaveMediaListEntry MediaListEntry `json:"SaveMediaListEntry"`
}

// AnimeListItem represents a simplified anime list item for display
type AnimeListItem struct {
	MediaID       int
	Title         string
	Progress      int
	EpisodesTotal int
	Score         float64
	CoverURL      string
	StartYear     int
	Status        string
}

