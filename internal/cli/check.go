package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Magnetkopf/pGallery/internal/model"
	"gopkg.in/yaml.v3"
)

type CheckArgs struct {
	Base   string
}

func Check(args CheckArgs) {
	downloadedRecordPath := filepath.Join(args.Base, "downloaded.json")

	fileContent, err := os.ReadFile(downloadedRecordPath)
	if err != nil {
		log.Fatalf("Failed to read downloaded.json: %v", err)
	}

	var artworkIDs []int
	if err := json.Unmarshal(fileContent, &artworkIDs); err != nil {
		log.Fatalf("Failed to parse downloaded.json: %v", err)
	}

	log.Printf("Checking %d artworks...", len(artworkIDs))

	var validIDs []int
	removed := 0

	for _, artworkID := range artworkIDs {
		// Find the artwork folder: base/<artistID>/<artworkID>
		matches, err := filepath.Glob(filepath.Join(args.Base, "*", strconv.Itoa(artworkID)))
		if err != nil || len(matches) == 0 {
			log.Printf("❌ Artwork %d: folder not found, removing from downloaded.json", artworkID)
			removed++
			continue
		}
		artworkPath := matches[0]

		// Read artwork.yaml for page count
		yamlBytes, err := os.ReadFile(filepath.Join(artworkPath, "artwork.yaml"))
		if err != nil {
			log.Printf("❌ Artwork %d: artwork.yaml missing (%v), removing", artworkID, err)
			_ = os.RemoveAll(artworkPath)
			removed++
			continue
		}

		var artworkData model.ArtworkData
		if err := yaml.Unmarshal(yamlBytes, &artworkData); err != nil {
			log.Printf("❌ Artwork %d: artwork.yaml parse error (%v), removing", artworkID, err)
			_ = os.RemoveAll(artworkPath)
			removed++
			continue
		}

		pageCount := artworkData.PageCount
		ok := true

		// Check folder.*
		folderMatches, _ := filepath.Glob(filepath.Join(artworkPath, "folder.*"))
		if len(folderMatches) == 0 {
			log.Printf("❌ Artwork %d: missing folder.* thumbnail", artworkID)
			ok = false
		}

		// Check p0.*, p1.*, …, p{pageCount-1}.*
		for i := 0; i < pageCount && ok; i++ {
			pageMatches, _ := filepath.Glob(filepath.Join(artworkPath, fmt.Sprintf("p%d.*", i)))
			if len(pageMatches) == 0 {
				log.Printf("❌ Artwork %d: missing p%d.* (%d pages expected)", artworkID, i, pageCount)
				ok = false
			}
		}

		if ok {
			log.Printf("✅ Artwork %d: OK (%d pages)", artworkID, pageCount)
			validIDs = append(validIDs, artworkID)
		} else {
			log.Printf("🗑️  Artwork %d: incomplete, deleting folder and removing from downloaded.json", artworkID)
			if err := os.RemoveAll(artworkPath); err != nil {
				log.Printf("⚠️  Failed to remove %s: %v", artworkPath, err)
			}
			removed++
		}
	}

	// Persist updated downloaded.json
	jsonData, err := json.MarshalIndent(validIDs, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal updated downloaded.json: %v", err)
	}
	if err := os.WriteFile(downloadedRecordPath, jsonData, 0644); err != nil {
		log.Fatalf("Failed to write updated downloaded.json: %v", err)
	}

	log.Printf("Check complete: %d OK, %d removed", len(validIDs), removed)
}
