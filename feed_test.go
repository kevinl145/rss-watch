package main

import (
	"strings"
	"testing"
)

const rssSample = `<?xml version="1.0"?>
<rss version="2.0">
  <channel>
    <title>  Example Blog  </title>
    <item>
      <title>Has a GUID</title>
      <link>https://example.com/posts/has-a-guid</link>
      <guid>urn:uuid:abc-123</guid>
      <pubDate>Sat, 22 Aug 2026 09:00:00 GMT</pubDate>
    </item>
    <item>
      <title>No GUID</title>
      <link>https://example.com/posts/no-guid</link>
      <pubDate>Sun, 23 Aug 2026 09:00:00 GMT</pubDate>
    </item>
  </channel>
</rss>`

const atomSample = `<?xml version="1.0"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Example Atom Blog</title>
  <entry>
    <id>tag:example.com,2026:1</id>
    <title>Published Entry</title>
    <published>2026-08-22T09:00:00Z</published>
    <updated>2026-08-22T10:00:00Z</updated>
    <link rel="self" href="https://example.com/feed.xml"/>
    <link rel="alternate" href="https://example.com/posts/published-entry"/>
  </entry>
  <entry>
    <id>tag:example.com,2026:2</id>
    <title>Updated Only, Bare Link</title>
    <updated>2026-08-23T10:00:00Z</updated>
    <link href="https://example.com/posts/updated-only"/>
  </entry>
</feed>`

const rdfSample = `<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
         xmlns="http://purl.org/rss/1.0/"
         xmlns:dc="http://purl.org/dc/elements/1.1/">
  <channel rdf:about="https://example.com/">
    <title>  Old Style Feed  </title>
    <link>https://example.com/</link>
  </channel>
  <item rdf:about="https://example.com/posts/has-about">
    <title>Has an About</title>
    <link>https://example.com/posts/has-about</link>
    <dc:date>2026-08-22T09:00:00Z</dc:date>
  </item>
  <item>
    <title>No About</title>
    <link>https://example.com/posts/no-about</link>
  </item>
</rdf:RDF>`

func TestParseFeedRSS(t *testing.T) {
	feed, err := parseFeed(strings.NewReader(rssSample))
	if err != nil {
		t.Fatalf("parseFeed: %v", err)
	}
	if feed.Title != "Example Blog" {
		t.Errorf("title = %q, want %q", feed.Title, "Example Blog")
	}
	if len(feed.Items) != 2 {
		t.Fatalf("got %d items, want 2", len(feed.Items))
	}

	first := feed.Items[0]
	if first.ID != "urn:uuid:abc-123" {
		t.Errorf("first item ID = %q, want guid", first.ID)
	}
	if first.Published != "Sat, 22 Aug 2026 09:00:00 GMT" {
		t.Errorf("first item Published = %q", first.Published)
	}

	second := feed.Items[1]
	if second.ID != "https://example.com/posts/no-guid" {
		t.Errorf("second item ID = %q, want fallback to link", second.ID)
	}
}

func TestParseFeedAtom(t *testing.T) {
	feed, err := parseFeed(strings.NewReader(atomSample))
	if err != nil {
		t.Fatalf("parseFeed: %v", err)
	}
	if feed.Title != "Example Atom Blog" {
		t.Errorf("title = %q, want %q", feed.Title, "Example Atom Blog")
	}
	if len(feed.Items) != 2 {
		t.Fatalf("got %d items, want 2", len(feed.Items))
	}

	first := feed.Items[0]
	if first.Link != "https://example.com/posts/published-entry" {
		t.Errorf("first item Link = %q, want the rel=alternate link, not rel=self", first.Link)
	}
	if first.Published != "2026-08-22T09:00:00Z" {
		t.Errorf("first item Published = %q, want <published> over <updated>", first.Published)
	}

	second := feed.Items[1]
	if second.Link != "https://example.com/posts/updated-only" {
		t.Errorf("second item Link = %q, want the bare (no rel) link", second.Link)
	}
	if second.Published != "2026-08-23T10:00:00Z" {
		t.Errorf("second item Published = %q, want fallback to <updated>", second.Published)
	}
}

func TestParseFeedRDF(t *testing.T) {
	feed, err := parseFeed(strings.NewReader(rdfSample))
	if err != nil {
		t.Fatalf("parseFeed: %v", err)
	}
	if feed.Title != "Old Style Feed" {
		t.Errorf("title = %q, want %q", feed.Title, "Old Style Feed")
	}
	if len(feed.Items) != 2 {
		t.Fatalf("got %d items, want 2", len(feed.Items))
	}

	first := feed.Items[0]
	if first.ID != "https://example.com/posts/has-about" {
		t.Errorf("first item ID = %q, want rdf:about", first.ID)
	}
	if first.Published != "2026-08-22T09:00:00Z" {
		t.Errorf("first item Published = %q, want dc:date", first.Published)
	}

	second := feed.Items[1]
	if second.ID != "https://example.com/posts/no-about" {
		t.Errorf("second item ID = %q, want fallback to link", second.ID)
	}
}

func TestParseFeedInvalidXML(t *testing.T) {
	_, err := parseFeed(strings.NewReader("not xml at all"))
	if err == nil {
		t.Fatal("parseFeed: expected an error for non-XML input, got nil")
	}
}
