package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Magnetkopf/pGallery/cmd"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "sync":
		syncCmd := flag.NewFlagSet("sync", flag.ExitOnError)
		flagCookieFile := syncCmd.String("cookie", "cookie.txt", "where is your cookie.txt")
		flagUser := syncCmd.String("user", "", "bookmarks' owner id to sync")
		flagBase := syncCmd.String("base", "downloads", "base directory to save artworks")

		syncCmd.Parse(os.Args[2:])

		if *flagUser == "" {
			fmt.Println("Error: -user is required")
			syncCmd.PrintDefaults()
			os.Exit(1)
		}

		cookieBytes, err := os.ReadFile(*flagCookieFile)
		if err != nil {
			fmt.Printf("Error reading cookie file: %v\n", err)
			os.Exit(1)
		}

		cmd.Sync(cmd.SyncArgs{
			UserID: *flagUser,
			Cookie: string(cookieBytes),
			Base:   *flagBase,
		})
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`pGallery

Usage:
  pGallery <command> [arguments]

Commands:
  sync    Sync bookmarks for a user

Use "pGallery sync -help" for more information about the sync command.
`)
}
