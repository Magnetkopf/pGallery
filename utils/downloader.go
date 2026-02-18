package utils

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

const WorkerCount = 8

type DownloaderArgs struct {
	Url      string
	SavePath string
	FileName string
	Referer  string
	Downloader string
}

// Download chooses aria2c or built-in downloader to download file
func Download(args DownloaderArgs) bool {
	if !checkAria2c() || args.Downloader == "built-in" {
		//aria2c not found or built-in downloader is specified
		err := simpleDownload(args)
		if err != nil {
			log.Printf("Failed to download %s: %v", args.Url, err)
			return false
		}
		return true
	}

	maxRetries := 5
	for attempts := 0; attempts < maxRetries; attempts++ {
		cmd := exec.Command("aria2c",
			"--allow-overwrite=true",
			"--referer", args.Referer,
			"-d", args.SavePath,
			"-o", args.FileName,
			args.Url,
		)
		cmd.Stdout = nil
		cmd.Stderr = nil

		err := cmd.Run()
		if err == nil {
			return true
		} else {
			log.Printf("Failed to download %s (attempt %d/%d), retrying in 1s... (%v)", args.Url, attempts+1, maxRetries, err)
			time.Sleep(1 * time.Second)
		}
	}
	log.Fatalf("Failed to download after %d attempts: %s", maxRetries, args.Url)
	return false
}

func checkAria2c() bool {
	cmd := exec.Command("aria2c", "--version")
	err := cmd.Run()
	if err != nil {
		log.Fatalf("aria2c not found: %v", err)
		return false
	}
	return true
}

// simpleDownload uses built-in downloader to download file
func simpleDownload(args DownloaderArgs) error {

	//send head request
	req, err := http.NewRequest("HEAD", args.Url, nil)
	if err != nil {
		return err
	}
	if args.Referer != "" {
		req.Header.Set("Referer", args.Referer)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("Bad status code: " + resp.Status)
	}

	//get file size
	contentLength := resp.Header.Get("Content-Length")
	fileSize, _ := strconv.ParseInt(contentLength, 10, 64)
	fmt.Printf("📦 File size: %.2f MB\n", float64(fileSize)/1024/1024)

	//check if support range
	if resp.Header.Get("Accept-Ranges") != "bytes" {
		return errors.New("Server does not support range")
	}

	//ensure directory exists
	if err := os.MkdirAll(args.SavePath, 0755); err != nil {
		return err
	}

	//create file
	outFile, err := os.Create(args.SavePath + "/" + args.FileName)
	if err != nil {
		return err
	}
	defer outFile.Close()

	//pre allocate file space
	if err := outFile.Truncate(fileSize); err != nil {
		return err
	}

	//calculate part size
	var wg sync.WaitGroup
	partSize := fileSize / int64(WorkerCount)

	//start multiple threads
	progressChan := make(chan int64, WorkerCount)
	for i := 0; i < WorkerCount; i++ {
		wg.Add(1)

		startByte := int64(i) * partSize
		endByte := startByte + partSize - 1

		//process the last part, it may have remaining bytes
		//example: 15=3*4+3, so the last part is for 3
		if i == WorkerCount-1 {
			endByte = fileSize - 1
		}

		//start download
		go func(id int, start, end int64) {
			defer wg.Done()
			downloadPart(id, args.Url, args.Referer, start, end, outFile, progressChan)
		}(i, startByte, endByte)
	}

	doneChan := make(chan bool)
	go func() {
		var totalDownloaded int64
		for n := range progressChan {
			totalDownloaded += n
			percent := float64(totalDownloaded) / float64(fileSize) * 100
			fmt.Printf("\r⏳ Downloading: %.2f%%", percent) //use \r to keep the same line
		}
		fmt.Println() //new line
		doneChan <- true
	}()

	//wait for all parts to finish
	wg.Wait()
	close(progressChan)
	<-doneChan

	return nil

}

// downloadPart downloads parts of the file
func downloadPart(id int, url string, referer string, start, end int64, file *os.File, progress chan<- int64) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	//set Range header
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error downloading part %d: %v\n", id, err)
		return
	}
	defer resp.Body.Close()

	//buffer
	buf := make([]byte, 32*1024) // 32KB
	var written int64 = 0

	for {
		nr, er := resp.Body.Read(buf)
		if nr > 0 {
			_, ew := file.WriteAt(buf[0:nr], start+written)
			if ew != nil {
				fmt.Printf("Error writing part %d: %v\n", id, ew)
				return
			}
			written += int64(nr)
			progress <- int64(nr)
		}
		if er == io.EOF {
			break
		}
		if er != nil {
			fmt.Printf("Error reading part %d: %v\n", id, er)
			return
		}
	}
	//fmt.Printf("Part %d completed\n", id)
}
