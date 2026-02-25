package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Magnetkopf/pGallery/internal/model"
	"github.com/Magnetkopf/pGallery/internal/pixiv"
	"github.com/Magnetkopf/pGallery/utils"
	"github.com/tidwall/gjson"
	"gopkg.in/yaml.v3"
)

type SyncArgs struct {
	Cookie     string
	UserID     string
	Base       string
	Downloader string
}

const limitPerPage = 48

func Sync(args SyncArgs) {
	client := &pixiv.Client{
		Cookie: args.Cookie,
	}

	utils.InitUI()
	defer utils.StopUI()

	downloadManager := utils.NewDownloadManager(5)
	defer downloadManager.Wait()

	// Ensure base directory exists
	if err := os.MkdirAll(args.Base, 0755); err != nil {
		log.Fatalf("Failed to create base directory: %v", err)
	}

	dest := fmt.Sprintf("https://www.pixiv.net/ajax/user/%s/illusts/bookmarks?tag=&offset=0&limit=%d&rest=show&lang=en", args.UserID, limitPerPage)
	res, err := client.Get(dest)
	if err != nil {
		log.Fatalln("Error fetching initial bookmarks:", err)
		return
	}

	if gjson.Get(res, "error").Bool() {
		log.Fatalf("API Error: %s", gjson.Get(res, "message").String())
	}

	totalArtworks := gjson.Get(res, "body.total").Int()
	totalPages := int((totalArtworks + limitPerPage - 1) / limitPerPage)
	log.Printf("Total artworks: %d, Total pages: %d", totalArtworks, totalPages)

	var artworkList []int
	artistPFP := make(map[int]string)

	downloadedRecordPath := filepath.Join(args.Base, "downloaded.json")
	downloadedMap := make(map[int]bool)

	if fileContent, err := os.ReadFile(downloadedRecordPath); err == nil {
		var loadedIDs []int
		if err := json.Unmarshal(fileContent, &loadedIDs); err == nil {
			for _, id := range loadedIDs {
				downloadedMap[id] = true
			}
			log.Printf("Loaded %d records from downloaded.json", len(loadedIDs))
		}
	}

	for i := 0; i < totalPages; i++ {
		offset := i * limitPerPage
		log.Printf("🔍 Fetching page %d/%d...", i+1, totalPages)

		dest = fmt.Sprintf("https://www.pixiv.net/ajax/user/%s/illusts/bookmarks?tag=&offset=%d&limit=%d&rest=show&lang=en", args.UserID, offset, limitPerPage)
		bookmarkRes, err := client.Get(dest)
		if err != nil {
			log.Printf("Error fetching page %d: %v", i, err)
			continue
		}

		gjson.Get(bookmarkRes, "body.works").ForEach(func(_, value gjson.Result) bool {
			artworkID := int(value.Get("id").Int())
			artistID := int(value.Get("userId").Int())

			artworkList = append(artworkList, artworkID)

			//replace to get higher quality profile photo
			artistPFP[artistID] = strings.Replace(value.Get("profileImageUrl").String(), "_50.", "_170.", -1)
			return true
		})

	}

	utils.UILog(fmt.Sprintf("Found %d artworks, Expect %d artworks", len(artworkList), totalArtworks))

	for _, artworkID := range artworkList {
		if downloadedMap[artworkID] {
			utils.UILog(fmt.Sprintf("\033[1;36m Skipped: %d \033[0m", artworkID))
			continue
		}

		dest = fmt.Sprintf("https://www.pixiv.net/ajax/illust/%d", artworkID)
		illustRes, err := client.Get(dest)
		if err != nil {
			log.Printf("Error fetching artwork %d: %v", artworkID, err)
			continue
		}

		if gjson.Get(illustRes, "error").Bool() {
			log.Printf("API Error for artwork %d: %s", artworkID, gjson.Get(illustRes, "message").String())
			continue
		}

		url := gjson.Get(illustRes, "body.urls.original").String()
		artistID := int(gjson.Get(illustRes, "body.userId").Int())
		artworkPath := filepath.Join(args.Base, strconv.Itoa(artistID), strconv.Itoa(int(artworkID)))
		artistPath := filepath.Join(args.Base, strconv.Itoa(artistID))
		artworkYamlFile := filepath.Join(artworkPath, "artwork.yaml")
		artistYamlFile := filepath.Join(artistPath, "artist.yaml")

		// Create folder
		if err := os.MkdirAll(artworkPath, 0755); err != nil {
			log.Printf("⚠️ Failed to create artwork directory: %v", err)
			continue
		}

		// Download all pictures
		pageCount := gjson.Get(illustRes, "body.pageCount").Uint()
		for i := uint64(0); i < pageCount; i++ { //download all pictures
			fileExtension := url[len(url)-3:] //file extension
			var fileName = "p" + strconv.Itoa(int(i)) + "." + fileExtension
			newUrl := strings.Replace(url, "_p0.", "_p"+strconv.Itoa(int(i))+".", -1)

			capI := i
			capFileName := fileName
			capFileExt := fileExtension
			capArtworkPath := artworkPath
			capTaskID := fmt.Sprintf("%d_%s", artworkID, fileName)

			downloadManager.Add(utils.DownloadTask{
				Args: utils.DownloaderArgs{
					ID:         capTaskID,
					Url:        newUrl,
					SavePath:   capArtworkPath,
					FileName:   capFileName,
					Referer:    "https://www.pixiv.net",
					Downloader: args.Downloader,
				},
				OnComplete: func(success bool) {
					if success {
						fullFilePath := filepath.Join(capArtworkPath, capFileName)
						err := utils.ModifyPictureExtension(fullFilePath)
						if err != nil {
							log.Printf("⚠️ Failed to modify picture extension: %v", err)
							return
						}

						if capI == 0 { //copy p0 as folder picture
							folderFileName := "folder." + capFileExt
							folderFilePath := filepath.Join(capArtworkPath, folderFileName)
							if err := utils.CopyFile(fullFilePath, folderFilePath); err != nil {
								log.Printf("⚠️ Failed to create folder image: %v", err)
							}
						}
					} else {
						log.Fatalf("failed to download %s", capFileName)
					}
				},
			})
		}
		utils.UILog(fmt.Sprintf("👀 %d", artworkID))

		// Download artist pfp
		artistPFPUrl := artistPFP[artistID]
		if artistPFPUrl != "" {
			capArtistPath := artistPath
			capArtistPFPUrl := artistPFPUrl
			capArtistTaskID := fmt.Sprintf("%d(pfp)", artistID)
			downloadManager.Add(utils.DownloadTask{
				Args: utils.DownloaderArgs{
					ID:         capArtistTaskID,
					Url:        capArtistPFPUrl,
					SavePath:   capArtistPath,
					FileName:   "folder.jpg",
					Referer:    "https://www.pixiv.net",
					Downloader: args.Downloader,
				},
				OnComplete: func(success bool) {
					if success {
						fullFilePath := filepath.Join(capArtistPath, "folder.jpg")
						err := utils.ModifyPictureExtension(fullFilePath)
						if err != nil {
							log.Printf("⚠️ Failed to modify picture extension: %v", err)
						}
					} else {
						log.Printf("⚠️ Failed to download artist pfp: %s", capArtistPFPUrl)
					}
				},
			})
		}

		// YAML files
		var tagData []model.TagData
		gjson.Get(illustRes, "body.tags.tags").ForEach(func(_, value gjson.Result) bool {
			tagData = append(tagData, model.TagData{
				Tag:         value.Get("tag").String(),
				Locked:      value.Get("locked").Bool(),
				Romaji:      value.Get("romaji").String(),
				Translation: value.Get("translation.en").String(),
			})
			return true
		})

		artworkDetailData := model.ArtworkData{
			ID:          int(gjson.Get(illustRes, "body.id").Int()),
			Title:       gjson.Get(illustRes, "body.title").String(),
			Description: gjson.Get(illustRes, "body.description").String(),
			PageCount:   int(pageCount),
			Tags:        tagData,
			OriginalUrl: gjson.Get(illustRes, "body.urls.original").String(),
			ArtistId:    artistID,
			ArtistName:  gjson.Get(illustRes, "body.userName").String(),
			CreateDate:  gjson.Get(illustRes, "body.createDate").String(),
		}

		artistDetailData := model.ArtistData{
			ID:      int(gjson.Get(illustRes, "body.userId").Int()),
			Name:    gjson.Get(illustRes, "body.userName").String(),
			Account: gjson.Get(illustRes, "body.userAccount").String(),
		}

		//write to FS
		artworkYamlBytes, err := yaml.Marshal(artworkDetailData)
		if err != nil {
			log.Fatalf("Error marshaling YAML: %v", err)
		}
		//overwrite if exists
		err = os.WriteFile(artworkYamlFile, artworkYamlBytes, 0644)
		if err != nil {
			log.Fatalf("Error writing YAML file: %v", err)
		}
		artistYamlBytes, err := yaml.Marshal(artistDetailData)
		if err != nil {
			log.Fatalf("Error marshaling YAML: %v", err)
		}
		//overwrite if exists
		err = os.WriteFile(artistYamlFile, artistYamlBytes, 0644)
		if err != nil {
			log.Fatalf("Error writing YAML file: %v", err)
		}

		//mark downloaded artwork
		downloadedMap[artworkID] = true

		//map -> slice
		var idsToWrite []int
		for id := range downloadedMap {
			idsToWrite = append(idsToWrite, id)
		}

		if jsonData, err := json.MarshalIndent(idsToWrite, "", "  "); err == nil {
			_ = os.WriteFile(downloadedRecordPath, jsonData, 0644)
		}

		time.Sleep(1 * time.Second) //wait 1s

	}
}
