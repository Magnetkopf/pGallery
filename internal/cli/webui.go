package cli

import (
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

type WebUIArgs struct {
	Base string
	Port int
}

type WebUIContext struct {
	Store *model.Store
	Base  string
}

func WebUI(args WebUIArgs) {
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

	ctx := &WebUIContext{
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

const layoutTmpl = `
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>pGallery WebUI</title>
	<style>
		body { font-family: sans-serif; margin: 0; padding: 20px; background: #f0f0f0; }
		header { background: #333; color: #fff; padding: 10px; margin-bottom: 20px; }
		header a { color: #fff; text-decoration: none; margin-right: 20px; }
		h1 { margin: 0 0 10px 0; font-size: 1.5em; }
		.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 20px; }
		.card { background: #fff; padding: 10px; border-radius: 5px; box-shadow: 0 2px 5px rgba(0,0,0,0.1); }
		.card img { max-width: 100%; height: auto; display: block; margin-bottom: 10px; }
		.card .title { font-weight: bold; margin-bottom: 5px; font-size: 0.9em; }
		.card .meta { color: #666; font-size: 0.8em; }
		.list-item { background: #fff; padding: 10px; margin-bottom: 5px; border-radius: 3px; }
		.list-item a { text-decoration: none; color: #333; }
		.list-item a:hover { text-decoration: underline; }
		.filter-info { margin-bottom: 20px; padding: 10px; background: #e0e0e0; border-radius: 4px; }
	</style>
</head>
<body>
	<header>
		<nav>
			<a href="/">All Artworks</a>
			<a href="/artist">Artists</a>
			<a href="/tag">Tags</a>
		</nav>
	</header>
	<main>
		{{template "content" .}}
	</main>
</body>
</html>
`

const homeTmpl = `
{{define "content"}}
	{{if .Filter}}
		<div class="filter-info">
			Filter: <strong>{{.Filter}}</strong>
		</div>
	{{end}}
	<div class="grid">
		{{range .Artworks}}
			<div class="card">
				<img src="/static/{{.Thumbnail}}" loading="lazy" alt="{{.Title}}">
				<a href="/artwork?id={{.ID}}">
					<div class="title">{{.Title}}</div>
				</a>
				<div class="meta">ID: {{.ID}}</div>
				<div class="meta">Pages: {{.PageCount}}</div>
				<div class="meta">Artist: <a href="/?artist={{.ArtistID}}">{{.ArtistID}}</a></div>
			</div>
		{{end}}
	</div>
{{end}}
`

const listTmpl = `
{{define "content"}}
	<h1>{{.Title}}</h1>
	<div class="list">
		{{range .Items}}
			<div class="list-item">
				<a href="/?{{$.Type}}={{.Value}}">{{.Label}} ({{.Count}})</a>
			</div>
		{{end}}
	</div>
{{end}}
`

// View Models

type HomeView struct {
	Artworks []*model.ArtworkCard
	Filter   string
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

func (ctx *WebUIContext) handleHome(w http.ResponseWriter, r *http.Request) {
	artistID := r.URL.Query().Get("artist")
	tagName := r.URL.Query().Get("tag")

	var artworks []*model.ArtworkCard
	var filterInfo string

	if artistID != "" {
		filterInfo = "Artist ID: " + artistID
		if detail, ok := ctx.Store.ArtistIndex[artistID]; ok {
			artworks = detail.Artworks
			if detail.Name != "" {
				filterInfo = "Artist: " + detail.Name
			}
		}
	} else if tagName != "" {
		filterInfo = "Tag: " + tagName
		artworks = ctx.Store.TagIndex[tagName]
	} else {
		artworks = make([]*model.ArtworkCard, 0, len(ctx.Store.ArtworkIndex))
		for _, artwork := range ctx.Store.ArtworkIndex {
			artworks = append(artworks, artwork)
		}
	}

	sort.Slice(artworks, func(i, j int) bool {
		return artworks[i].ID > artworks[j].ID
	})

	view := HomeView{
		Artworks: artworks,
		Filter:   filterInfo,
	}

	tmpl, err := template.New("layout").Parse(layoutTmpl)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	tmpl, err = tmpl.Parse(homeTmpl)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	err = tmpl.Execute(w, view)
	if err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

func (ctx *WebUIContext) handleArtistList(w http.ResponseWriter, r *http.Request) {
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

func (ctx *WebUIContext) handleTagList(w http.ResponseWriter, r *http.Request) {
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
	tmpl, err := template.New("layout").Parse(layoutTmpl)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	tmpl, err = tmpl.Parse(listTmpl)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	err = tmpl.Execute(w, view)
	if err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

const artworkTmpl = `
{{define "content"}}
	<div class="artwork-detail">
		<h1>{{.Artwork.Title}}</h1>
		<div class="meta">
			Artist: <a href="/?artist={{.Artwork.ArtistId}}">{{.Artwork.ArtistName}}</a> |
			Date: {{.Artwork.CreateDate}} |
			Pages: {{.Artwork.PageCount}}
		</div>
		<div class="tags">
			Tags:
			{{range .Artwork.Tags}}
				<a href="/?tag={{.Tag}}">{{.Tag}}</a>
			{{end}}
		</div>
		<div class="description">
			{{.Artwork.Description}}
		</div>
		<div class="images">
			{{range .Images}}
				<div class="image-container">
					<img src="/static/{{.}}" alt="Page">
				</div>
			{{end}}
		</div>
	</div>
	<style>
		.artwork-detail { background: #fff; padding: 20px; border-radius: 5px; }
		.artwork-detail .meta { margin-bottom: 10px; color: #666; }
		.artwork-detail .tags a { margin-right: 10px; text-decoration: none; color: #007bff; }
		.artwork-detail .description { margin: 20px 0; white-space: pre-wrap; }
		.image-container { margin-bottom: 20px; text-align: center; }
		.image-container img { max-width: 100%; height: auto; box-shadow: 0 2px 5px rgba(0,0,0,0.1); }
	</style>
{{end}}
`

func (ctx *WebUIContext) handleArtwork(w http.ResponseWriter, r *http.Request) {
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

	tmpl, err := template.New("layout").Parse(layoutTmpl)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	tmpl, err = tmpl.Parse(artworkTmpl)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	err = tmpl.Execute(w, view)
	if err != nil {
		log.Printf("Error executing template: %v", err)
	}
}
