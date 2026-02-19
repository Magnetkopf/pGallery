package web

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Magnetkopf/pGallery/internal/model"
	"gopkg.in/yaml.v3"
)

//go:embed templates/*.html
var templateFS embed.FS

type ServerArgs struct {
	Base string
	Port int
}

type WebContext struct {
	Store *model.Store
	Base  string
}

func Start(args ServerArgs) {
	fmt.Printf("Starting Web UI on port %d with base %s\n", args.Port, args.Base)

	// Load index.json
	indexPath := filepath.Join(args.Base, "index.json")
	fileContent, err := os.ReadFile(indexPath)
	if err != nil {
		log.Fatalf("Failed to read index.json: %v", err)
	}

	var store model.Store
	if err := json.Unmarshal(fileContent, &store); err != nil {
		log.Fatalf("Failed to parse index.json: %v", err)
	}

	ctx := &WebContext{
		Store: &store,
		Base:  args.Base,
	}

	// Handlers
	http.HandleFunc("/", ctx.handleHome)
	http.HandleFunc("/artist", ctx.handleArtistList)
	http.HandleFunc("/tag", ctx.handleTagList)
	http.HandleFunc("/artwork", ctx.handleArtwork)

	fs := http.FileServer(http.Dir(args.Base))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	addr := fmt.Sprintf(":%d", args.Port)
	log.Printf("Listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

type Pagination struct {
	CurrentPage int
	TotalPages  int
	TotalItems  int
	Limit       int
	HasPrev     bool
	HasNext     bool
	PrevPage    int
	NextPage    int
}

type HomeView struct {
	Artworks   []*model.ArtworkCard
	Filter     string
	Query      template.URL
	Pagination Pagination
}

type ListItem struct {
	Label string
	Value string
	Count int
}

type ListView struct {
	Title string
	Type  string // "artist" or "tag" -> used for query param key
	Items []ListItem
}

type ArtworkDetailView struct {
	Artwork model.ArtworkData
	Images  []string
}

// Handlers Implementation

func (ctx *WebContext) handleHome(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	artistID := query.Get("artist")
	tagName := query.Get("tag")
	pageStr := query.Get("page")
	limitStr := query.Get("limit")

	page := 1
	if pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
	}
	if page < 1 {
		page = 1
	}

	limit := 20
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}
	if limit < 1 {
		limit = 20
	}

	// Filter
	var filtered []*model.ArtworkCard

	// If no filters, start with all
	if artistID == "" && tagName == "" {
		filtered = make([]*model.ArtworkCard, 0, len(ctx.Store.ArtworkIndex))
		for _, artwork := range ctx.Store.ArtworkIndex {
			filtered = append(filtered, artwork)
		}
	} else {
		// If artist filter is present
		var artistArtworks []*model.ArtworkCard
		if artistID != "" {
			if detail, ok := ctx.Store.ArtistIndex[artistID]; ok {
				artistArtworks = detail.Artworks
			}
		}

		// If tag filter is present
		var tagArtworks []*model.ArtworkCard
		if tagName != "" {
			tagArtworks = ctx.Store.TagIndex[tagName]
		}

		// Combine filters
		if artistID != "" && tagName != "" {

			tagMap := make(map[string]bool)
			for _, art := range tagArtworks {
				tagMap[art.ID] = true
			}
			for _, art := range artistArtworks {
				if tagMap[art.ID] {
					filtered = append(filtered, art)
				}
			}
		} else if artistID != "" {
			filtered = artistArtworks
		} else if tagName != "" {
			filtered = tagArtworks
		}
	}

	// Sort (descending by ID)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].ID > filtered[j].ID
	})

	// Pagination
	totalItems := len(filtered)
	totalPages := (totalItems + limit - 1) / limit
	if page > totalPages && totalPages > 0 {
		page = totalPages
	}

	start := (page - 1) * limit
	end := start + limit
	if start > totalItems {
		start = totalItems
	}
	if end > totalItems {
		end = totalItems
	}

	pagedArtworks := filtered[start:end]

	var filterInfo []string
	if artistID != "" {
		filterInfo = append(filterInfo, "Artist: "+artistID)
		if detail, ok := ctx.Store.ArtistIndex[artistID]; ok && detail.Name != "" {
			filterInfo[len(filterInfo)-1] = "Artist: " + detail.Name
		}
	}
	if tagName != "" {
		filterInfo = append(filterInfo, "Tag: "+tagName)
	}

	// Reconstruct query for pagination links (excluding page and limit)
	q := r.URL.Query()
	q.Del("page")
	q.Del("limit")

	rawQuery := q.Encode()

	view := HomeView{
		Artworks: pagedArtworks,
		Filter:   strings.Join(filterInfo, ", "),
		Query:    template.URL(rawQuery),
		Pagination: Pagination{
			CurrentPage: page,
			TotalPages:  totalPages,
			TotalItems:  totalItems,
			Limit:       limit,
			HasPrev:     page > 1,
			HasNext:     page < totalPages,
			PrevPage:    page - 1,
			NextPage:    page + 1,
		},
	}

	tmpl, err := template.ParseFS(templateFS, "templates/layout.html", "templates/home.html")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	err = tmpl.Execute(w, view)
	if err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

func (ctx *WebContext) handleArtistList(w http.ResponseWriter, r *http.Request) {
	var items []ListItem
	for artistID, detail := range ctx.Store.ArtistIndex {
		label := detail.Name
		if label == "" {
			label = artistID
		}
		items = append(items, ListItem{
			Label: label,
			Value: artistID,
			Count: len(detail.Artworks),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Count > items[j].Count // Sort by count desc
	})

	view := ListView{
		Title: "Artists",
		Type:  "artist",
		Items: items,
	}

	renderList(w, view)
}

func (ctx *WebContext) handleTagList(w http.ResponseWriter, r *http.Request) {
	var items []ListItem
	for tag, artworks := range ctx.Store.TagIndex {
		items = append(items, ListItem{
			Label: tag,
			Value: tag,
			Count: len(artworks),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Count > items[j].Count
	})

	view := ListView{
		Title: "Tags",
		Type:  "tag",
		Items: items,
	}

	renderList(w, view)
}

func renderList(w http.ResponseWriter, view ListView) {
	tmpl, err := template.ParseFS(templateFS, "templates/layout.html", "templates/list.html")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	err = tmpl.Execute(w, view)
	if err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

func (ctx *WebContext) handleArtwork(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing id", http.StatusBadRequest)
		return
	}

	card, ok := ctx.Store.ArtworkIndex[id]
	if !ok {
		http.Error(w, "Artwork not found", http.StatusNotFound)
		return
	}

	artistPath := filepath.Join(ctx.Base, card.ArtistID)
	artworkPath := filepath.Join(artistPath, card.ID)
	artworkYamlPath := filepath.Join(artworkPath, "artwork.yaml")

	yamlBytes, err := os.ReadFile(artworkYamlPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read artwork.yaml: %v", err), http.StatusInternalServerError)
		return
	}

	var artworkData model.ArtworkData
	if err := yaml.Unmarshal(yamlBytes, &artworkData); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse artwork.yaml: %v", err), http.StatusInternalServerError)
		return
	}

	// Find images
	var images []string
	files, err := os.ReadDir(artworkPath)
	if err != nil {
		log.Printf("Failed to read dir %s: %v", artworkPath, err)
	} else {

		for i := 0; i < artworkData.PageCount; i++ {
			prefix := fmt.Sprintf("p%d.", i)
			found := false
			for _, file := range files {
				if strings.HasPrefix(file.Name(), prefix) {
					// Relative path for static file server: ArtistID/ArtworkID/Filename
					relPath := filepath.Join(card.ArtistID, card.ID, file.Name())
					images = append(images, relPath)
					found = true
					break
				}
			}
			if !found {
				log.Printf("Warning: Image for page %d not found in %s", i, artworkPath)
			}
		}
	}

	view := ArtworkDetailView{
		Artwork: artworkData,
		Images:  images,
	}

	tmpl, err := template.ParseFS(templateFS, "templates/layout.html", "templates/artwork.html")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	err = tmpl.Execute(w, view)
	if err != nil {
		log.Printf("Error executing template: %v", err)
	}
}
