package cmd

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Magnetkopf/pGallery/model"
	"gopkg.in/yaml.v3"
)

type BuildArgs struct {
	Base string
}

func Build(args BuildArgs) {
	log.Println("Building index...")

	store := model.Store{
		ArtworkIndex: make(map[string]*model.ArtworkCard),
		TagIndex:     make(map[string][]*model.ArtworkCard),
		ArtistIndex:  make(map[string][]*model.ArtworkCard),
	}

	artistEntries, err := os.ReadDir(args.Base)
	if err != nil {
		log.Fatalf("Failed to read base directory: %v", err)
	}

	for _, artistEntry := range artistEntries {
		if !artistEntry.IsDir() {
			continue
		}

		artistID := artistEntry.Name()
		artistPath := filepath.Join(args.Base, artistEntry.Name())

		artistYamlPath := filepath.Join(artistPath, "artist.yaml")
		if _, err := os.Stat(artistYamlPath); err != nil {
			log.Printf("Warning: No artist.yaml found for %s", artistID)
		}

		artworkEntries, err := os.ReadDir(artistPath)
		if err != nil {
			log.Printf("Failed to read artist directory %s: %v", artistPath, err)
			continue
		}

		for _, artworkEntry := range artworkEntries {
			if !artworkEntry.IsDir() {
				continue
			}

			artworkID := artworkEntry.Name()
			artworkPath := filepath.Join(artistPath, artworkEntry.Name())
			artworkYamlPath := filepath.Join(artworkPath, "artwork.yaml")

			artworkDataBytes, err := os.ReadFile(artworkYamlPath)
			if err != nil {
				log.Printf("Warning: Failed to read artwork.yaml for %s: %v", artworkID, err)
				continue
			}

			var artworkData model.ArtworkData
			if err := yaml.Unmarshal(artworkDataBytes, &artworkData); err != nil {
				log.Printf("Error unmarshaling artwork.yaml for %s: %v", artworkID, err)
				continue
			}

			var thumbnailPath string
			files, _ := os.ReadDir(artworkPath)
			for _, file := range files {
				if strings.HasPrefix(file.Name(), "p0.") { //use folder. is also okay
					thumbnailPath = filepath.Join(artistEntry.Name(), artworkEntry.Name(), file.Name())
					break
				}
			}

			card := &model.ArtworkCard{
				ID:        artworkID,
				ArtistID:  artistID,
				Title:     artworkData.Title,
				PageCount: artworkData.PageCount,
				Thumbnail: thumbnailPath,
			}

			store.ArtworkIndex[card.ID] = card
			store.ArtistIndex[artistID] = append(store.ArtistIndex[artistID], card)

			for _, tag := range artworkData.Tags {
				store.TagIndex[tag.Tag] = append(store.TagIndex[tag.Tag], card)
			}
		}
	}

	store.LastIndexed = time.Now()

	indexPath := filepath.Join(args.Base, "index.json")
	indexBytes, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal index: %v", err)
	}

	if err := os.WriteFile(indexPath, indexBytes, 0644); err != nil {
		log.Fatalf("Failed to write index.json: %v", err)
	}

	log.Printf("Indexed %d artworks.", len(store.ArtworkIndex))
}
