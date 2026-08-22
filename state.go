package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// State is keyed by feed URL so one file can track any number of feeds.
type State struct {
	Feeds map[string]FeedState `json:"feeds"`
}

type FeedState struct {
	Seen        map[string]bool `json:"seen"`
	LastChecked time.Time       `json:"last_checked"`
}

func loadState(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &State{Feeds: make(map[string]FeedState)}, nil
	}
	if err != nil {
		return nil, err
	}
	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, err
	}
	if st.Feeds == nil {
		st.Feeds = make(map[string]FeedState)
	}
	return &st, nil
}

func saveState(path string, st *State) error {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
