package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type output struct {
	Feed  string `json:"feed"`
	Title string `json:"title"`
	New   []Item `json:"new_items"`
}

type feedListEntry struct {
	Feed        string `json:"feed"`
	ItemsSeen   int    `json:"items_seen"`
	LastChecked string `json:"last_checked,omitempty"`
}

func main() {
	jsonOut := flag.Bool("json", false, "output new items as JSON instead of plain text")
	statePath := flag.String("state", defaultStatePath(), "path to the state file used to track seen items")
	timeout := flag.Duration("timeout", 15*time.Second, "HTTP timeout when fetching the feed")
	quiet := flag.Bool("quiet", false, "record current items as seen without printing anything (use on first run)")
	list := flag.Bool("list", false, "list feeds tracked in the state file and exit")
	configPath := flag.String("config", "", "path to a file listing feed URLs to watch, one per line, instead of a single feed argument")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [flags] <feed-url>\n       %s [flags] --config <path>\n\n", os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *list {
		st, err := loadState(*statePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "load state: %v\n", err)
			os.Exit(1)
		}
		if *jsonOut {
			printListJSON(st)
		} else {
			printList(st)
		}
		return
	}

	var feedURLs []string
	if *configPath != "" {
		if flag.NArg() != 0 {
			fmt.Fprintln(os.Stderr, "cannot use --config together with a feed URL argument")
			os.Exit(2)
		}
		urls, err := readConfig(*configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read config: %v\n", err)
			os.Exit(1)
		}
		if len(urls) == 0 {
			fmt.Fprintf(os.Stderr, "%s: no feed URLs found\n", *configPath)
			os.Exit(1)
		}
		feedURLs = urls
	} else {
		if flag.NArg() != 1 {
			flag.Usage()
			os.Exit(2)
		}
		feedURLs = []string{flag.Arg(0)}
	}

	st, err := loadState(*statePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load state: %v\n", err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: *timeout}
	showFeed := len(feedURLs) > 1
	exitCode := 0

	for _, feedURL := range feedURLs {
		title, newItems, err := fetchAndDiff(client, st, feedURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", feedURL, err)
			exitCode = 1
			continue
		}
		if !*quiet {
			if *jsonOut {
				printJSON(feedURL, title, newItems)
			} else {
				printText(feedURL, title, newItems, showFeed)
			}
		}
	}

	if err := saveState(*statePath, st); err != nil {
		fmt.Fprintf(os.Stderr, "save state: %v\n", err)
		os.Exit(1)
	}

	os.Exit(exitCode)
}

// fetchAndDiff fetches a single feed, works out which items haven't been
// seen before, and updates st in place with the new seen set, ETag,
// Last-Modified, and last-checked time. State is only updated on success,
// so a feed that fails to fetch or parse is left untouched for the next run.
func fetchAndDiff(client *http.Client, st *State, feedURL string) (title string, newItems []Item, err error) {
	fs := st.Feeds[feedURL]
	if fs.Seen == nil {
		fs.Seen = make(map[string]bool)
	}

	req, err := http.NewRequest(http.MethodGet, feedURL, nil)
	if err != nil {
		return "", nil, fmt.Errorf("fetch: %w", err)
	}
	if fs.ETag != "" {
		req.Header.Set("If-None-Match", fs.ETag)
	}
	if fs.LastModified != "" {
		req.Header.Set("If-Modified-Since", fs.LastModified)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotModified:
		// Server confirmed nothing changed; nothing new to report, and
		// the seen set doesn't need touching.
	case http.StatusOK:
		feed, err := parseFeed(resp.Body)
		if err != nil {
			return "", nil, fmt.Errorf("parse: %w", err)
		}
		title = feed.Title

		for _, item := range feed.Items {
			if !fs.Seen[item.ID] {
				newItems = append(newItems, item)
			}
		}
		for _, item := range feed.Items {
			fs.Seen[item.ID] = true
		}
		fs.ETag = resp.Header.Get("ETag")
		fs.LastModified = resp.Header.Get("Last-Modified")
	default:
		return "", nil, fmt.Errorf("unexpected status %s", resp.Status)
	}

	fs.LastChecked = time.Now()
	st.Feeds[feedURL] = fs
	return title, newItems, nil
}

func defaultStatePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".rss-watch-state.json"
	}
	return filepath.Join(home, ".rss-watch", "state.json")
}

func printText(feedURL, feedTitle string, items []Item, showFeed bool) {
	if showFeed {
		fmt.Printf("%s:\n", feedURL)
	}
	if len(items) == 0 {
		fmt.Println("no new items")
		return
	}
	fmt.Printf("%d new item(s) in %s:\n", len(items), feedTitle)
	for _, item := range items {
		fmt.Printf("- %s\n  %s\n", item.Title, item.Link)
	}
}

func sortedFeedURLs(st *State) []string {
	urls := make([]string, 0, len(st.Feeds))
	for u := range st.Feeds {
		urls = append(urls, u)
	}
	sort.Strings(urls)
	return urls
}

func printList(st *State) {
	urls := sortedFeedURLs(st)
	if len(urls) == 0 {
		fmt.Println("no feeds tracked yet")
		return
	}
	for _, u := range urls {
		fs := st.Feeds[u]
		lastChecked := "never"
		if !fs.LastChecked.IsZero() {
			lastChecked = fs.LastChecked.Format(time.RFC3339)
		}
		fmt.Printf("%s\n  %d item(s) seen, last checked %s\n", u, len(fs.Seen), lastChecked)
	}
}

func printListJSON(st *State) {
	urls := sortedFeedURLs(st)
	entries := make([]feedListEntry, 0, len(urls))
	for _, u := range urls {
		fs := st.Feeds[u]
		entry := feedListEntry{Feed: u, ItemsSeen: len(fs.Seen)}
		if !fs.LastChecked.IsZero() {
			entry.LastChecked = fs.LastChecked.Format(time.RFC3339)
		}
		entries = append(entries, entry)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(entries)
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
