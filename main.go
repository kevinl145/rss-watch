package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type output struct {
	Feed  string `json:"feed"`
	Title string `json:"title"`
	New   []Item `json:"new_items"`
}

func main() {
	jsonOut := flag.Bool("json", false, "output new items as JSON instead of plain text")
	statePath := flag.String("state", defaultStatePath(), "path to the state file used to track seen items")
	timeout := flag.Duration("timeout", 15*time.Second, "HTTP timeout when fetching the feed")
	quiet := flag.Bool("quiet", false, "record current items as seen without printing anything (use on first run)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [flags] <feed-url>\n\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	feedURL := flag.Arg(0)

	client := &http.Client{Timeout: *timeout}
	resp, err := client.Get(feedURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch %s: %v\n", feedURL, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "fetch %s: unexpected status %s\n", feedURL, resp.Status)
		os.Exit(1)
	}

	feed, err := parseFeed(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse %s: %v\n", feedURL, err)
		os.Exit(1)
	}

	st, err := loadState(*statePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load state: %v\n", err)
		os.Exit(1)
	}

	fs := st.Feeds[feedURL]
	if fs.Seen == nil {
		fs.Seen = make(map[string]bool)
	}

	var newItems []Item
	for _, item := range feed.Items {
		if !fs.Seen[item.ID] {
			newItems = append(newItems, item)
		}
	}

	if !*quiet {
		if *jsonOut {
			printJSON(feedURL, feed.Title, newItems)
		} else {
			printText(feed.Title, newItems)
		}
	}

	for _, item := range feed.Items {
		fs.Seen[item.ID] = true
	}
	fs.LastChecked = time.Now()
	st.Feeds[feedURL] = fs

	if err := saveState(*statePath, st); err != nil {
		fmt.Fprintf(os.Stderr, "save state: %v\n", err)
		os.Exit(1)
	}
}

func defaultStatePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".rss-watch-state.json"
	}
	return filepath.Join(home, ".rss-watch", "state.json")
}

func printText(feedTitle string, items []Item) {
	if len(items) == 0 {
		fmt.Println("no new items")
		return
	}
	fmt.Printf("%d new item(s) in %s:\n", len(items), feedTitle)
	for _, item := range items {
		fmt.Printf("- %s\n  %s\n", item.Title, item.Link)
	}
}

func printJSON(feedURL, title string, items []Item) {
	out := output{Feed: feedURL, Title: title, New: items}
	if out.New == nil {
		out.New = []Item{}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(out)
}
