package utils

import (
	"log"
	"os/exec"
	"time"
)

type Aria2cArgs struct {
	Url      string
	SavePath string
	FileName string
	Referer  string
}

func Download(args Aria2cArgs) bool {
	if !checkAria2c() {
		log.Fatalf("aria2c not found")
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