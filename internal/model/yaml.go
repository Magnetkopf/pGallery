package model

type TagData struct {
	Tag         string `yaml:"tag"`
	Locked      bool   `yaml:"locked"`
	Romaji      string `yaml:"romaji"`
	Translation string `yaml:"translation"`
}

type ArtworkData struct {
	ID          int       `yaml:"id"`
	Title       string    `yaml:"title"`
	Description string    `yaml:"description"`
	PageCount   int       `yaml:"pages"`
	Tags        []TagData `yaml:"tags"`
	OriginalUrl string    `yaml:"original_url"`
	ArtistId    int       `yaml:"artist_id"`
	ArtistName  string    `yaml:"artist_name"`
	CreateDate  string    `yaml:"create_date"`
}

type ArtistData struct {
	ID      int    `yaml:"id"`
	Name    string `yaml:"name"`
	Account string `yaml:"account"`
}
