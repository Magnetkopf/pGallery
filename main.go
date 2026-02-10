package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

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
			Cookie: strings.TrimSpace(string(cookieBytes)),
			Base:   *flagBase,
		})

	case "build":
		buildCmd := flag.NewFlagSet("build", flag.ExitOnError)
		flagBase := buildCmd.String("base", "downloads", "base directory to scan")

		buildCmd.Parse(os.Args[2:])

		if *flagBase == "" {
			fmt.Println("Error: -base is required")
			buildCmd.PrintDefaults()
			os.Exit(1)
		}

		cmd.Build(cmd.BuildArgs{
			Base: *flagBase,
		})

	case "webui":
		webuiCmd := flag.NewFlagSet("webui", flag.ExitOnError)
		flagBase := webuiCmd.String("base", "downloads", "base directory")
		flagPort := webuiCmd.Int("port", 8080, "port to listen on")

		webuiCmd.Parse(os.Args[2:])

		cmd.WebUI(cmd.WebUIArgs{
			Base: *flagBase,
			Port: *flagPort,
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
  sync      Sync bookmarks for a user
  build	  	Indexing the database
  webui     Start web UI

Use "pGallery <command> -help" for more information.
`)
}
