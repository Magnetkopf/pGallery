package utils

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	uiMu          sync.Mutex
	activeTasks   []string
	taskProgress  map[string]float64
	logs          []string
	uiTicker      *time.Ticker
	uiStop        chan struct{}
	linesRendered int
)

// LogInterceptor redirects log outputs to UILog
type LogInterceptor struct{}

func (l *LogInterceptor) Write(p []byte) (n int, err error) {
	// Strip trailing newline as UILog adds it when rendering
	msg := string(bytes.TrimSuffix(p, []byte("\n")))
	UILog(msg)
	return len(p), nil
}

// InitUI starts the periodic UI renderer
func InitUI() {
	activeTasks = make([]string, 0)
	taskProgress = make(map[string]float64)
	logs = make([]string, 0)
	uiStop = make(chan struct{})
	uiTicker = time.NewTicker(200 * time.Millisecond)

	// Redirect standard logger to our UI manager
	log.SetOutput(&LogInterceptor{})

	go func() {
		for {
			select {
			case <-uiTicker.C:
				renderUI()
			case <-uiStop:
				renderUI()
				return
			}
		}
	}()
}

// StopUI stops the periodic renderer
func StopUI() {
	if uiTicker != nil {
		uiTicker.Stop()
	}
	if uiStop != nil {
		close(uiStop)
	}
	log.SetOutput(os.Stderr)
}

// UILog appends a scrolling log message
func UILog(msg string) {
	uiMu.Lock()
	defer uiMu.Unlock()
	logs = append(logs, msg)
}

// UIAddDownload registers a new download task
func UIAddDownload(id string) {
	uiMu.Lock()
	defer uiMu.Unlock()
	for _, existing := range activeTasks {
		if existing == id {
			return // already exists
		}
	}
	activeTasks = append(activeTasks, id)
	taskProgress[id] = 0.0
}

// UIUpdateDownload updates the progress for a download task
func UIUpdateDownload(id string, percent float64) {
	uiMu.Lock()
	defer uiMu.Unlock()
	taskProgress[id] = percent
}

// UIRemoveDownload removes a download task from the active list
func UIRemoveDownload(id string) {
	uiMu.Lock()
	defer uiMu.Unlock()
	for i, t := range activeTasks {
		if t == id {
			activeTasks = append(activeTasks[:i], activeTasks[i+1:]...)
			break
		}
	}
	delete(taskProgress, id)
}

func renderUI() {
	uiMu.Lock()
	defer uiMu.Unlock()

	// Clear previously rendered fixed lines
	if linesRendered > 0 {
		fmt.Printf("\033[%dA\033[J", linesRendered)
	}

	// Print scrolling logs
	for _, l := range logs {
		fmt.Println(l)
	}
	logs = logs[:0]

	lines := 0
	if len(activeTasks) > 0 {
		fmt.Println("\nDownloading:")
		lines += 2
		for _, id := range activeTasks {
			percent := taskProgress[id]
			bars := int(percent / 10)
			if bars < 0 {
				bars = 0
			}
			if bars > 10 {
				bars = 10
			}
			barStr := strings.Repeat("=", bars)
			if bars < 10 {
				barStr += ">"
				barStr += strings.Repeat(" ", 10-bars-1)
			}

			// Format ID to a fixed width or trim
			displayID := id
			if len(displayID) > 20 {
				displayID = "..." + displayID[len(displayID)-17:]
			}

			fmt.Printf("%20s: [%s] %5.1f%%\n", displayID, barStr, percent)
			lines++
		}
	}

	linesRendered = lines
}
