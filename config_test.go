package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "feeds.txt")
	content := "# my feeds\n" +
		"https://example.com/a.xml\n" +
		"\n" +
		"  https://example.com/b.xml  \n" +
		"https://example.com/a.xml\n" +
		"# https://example.com/commented-out.xml\n" +
		"https://example.com/c.xml\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	urls, err := readConfig(path)
	if err != nil {
		t.Fatalf("readConfig: %v", err)
	}

	want := []string{
		"https://example.com/a.xml",
		"https://example.com/b.xml",
		"https://example.com/c.xml",
	}
	if len(urls) != len(want) {
		t.Fatalf("got %d urls, want %d: %v", len(urls), len(want), urls)
	}
	for i, u := range want {
		if urls[i] != u {
			t.Errorf("urls[%d] = %q, want %q", i, urls[i], u)
		}
	}
}

func TestReadConfigEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "feeds.txt")
	if err := os.WriteFile(path, []byte("# nothing but comments\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	urls, err := readConfig(path)
	if err != nil {
		t.Fatalf("readConfig: %v", err)
	}
	if len(urls) != 0 {
		t.Errorf("got %v, want no urls", urls)
	}
}

func TestReadConfigMissingFile(t *testing.T) {
	_, err := readConfig(filepath.Join(t.TempDir(), "does-not-exist.txt"))
	if err == nil {
		t.Fatal("readConfig: expected error for missing file, got nil")
	}
}
