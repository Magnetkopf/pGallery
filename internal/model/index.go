package model

import "time"

type ArtworkCard struct {
	ID        string `json:"id"`
	ArtistID  string `json:"artist_id"`
	Title     string `json:"title"`
	PageCount int    `json:"page_count"`
	Thumbnail string `json:"thumbnail"`
}

type ArtistDetail struct {
	Name     string         `json:"name"`
	Artworks []*ArtworkCard `json:"artworks"`
}

// use string for key seems not the best practice bruh
type Store struct {
	ArtworkIndex map[string]*ArtworkCard   `json:"artwork_index"`
	TagIndex     map[string][]*ArtworkCard `json:"tag_index"`
	ArtistIndex  map[string]*ArtistDetail  `json:"artist_index"`

	LastIndexed time.Time
}
