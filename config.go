package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// readConfig reads feed URLs from a config file, one per line. Blank lines
// and lines starting with # are ignored. Duplicate URLs are dropped, keeping
// the order they first appear in, so a feed listed twice isn't fetched twice.
func readConfig(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	seen := make(map[string]bool)
	var urls []string

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if seen[line] {
			continue
		}
		seen[line] = true
		urls = append(urls, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("line %d: %w", lineNum, err)
	}
	return urls, nil
}
