package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// Feed is our own flattened shape, independent of whether the source
// was RSS or Atom.
type Feed struct {
	Title string
	Items []Item
}

type Item struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Link      string `json:"link"`
	Published string `json:"published,omitempty"`
}

type rssDoc struct {
	Channel struct {
		Title string    `xml:"title"`
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	GUID    string `xml:"guid"`
	PubDate string `xml:"pubDate"`
}

type rdfDoc struct {
	Channel struct {
		Title string `xml:"title"`
	} `xml:"channel"`
	Items []rdfItem `xml:"item"`
}

type rdfItem struct {
	About string `xml:"about,attr"`
	Title string `xml:"title"`
	Link  string `xml:"link"`
	Date  string `xml:"date"`
}

type atomDoc struct {
	Title   string      `xml:"title"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	ID        string `xml:"id"`
	Title     string `xml:"title"`
	Updated   string `xml:"updated"`
	Published string `xml:"published"`
	Links     []struct {
		Href string `xml:"href,attr"`
		Rel  string `xml:"rel,attr"`
	} `xml:"link"`
}

func parseFeed(r io.Reader) (*Feed, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	root, err := rootElementName(data)
	if err != nil {
		return nil, err
	}

	switch root {
	case "rss":
		var doc rssDoc
		if err := xml.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("decode rss: %w", err)
		}
		feed := &Feed{Title: strings.TrimSpace(doc.Channel.Title)}
		for _, it := range doc.Channel.Items {
			id := it.GUID
			if id == "" {
				id = it.Link
			}
			feed.Items = append(feed.Items, Item{
				ID:        id,
				Title:     strings.TrimSpace(it.Title),
				Link:      strings.TrimSpace(it.Link),
				Published: strings.TrimSpace(it.PubDate),
			})
		}
		return feed, nil

	case "feed":
		var doc atomDoc
		if err := xml.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("decode atom: %w", err)
		}
		feed := &Feed{Title: strings.TrimSpace(doc.Title)}
		for _, e := range doc.Entries {
			link := ""
			for _, l := range e.Links {
				if l.Rel == "" || l.Rel == "alternate" {
					link = l.Href
					break
				}
			}
			published := e.Published
			if published == "" {
				published = e.Updated
			}
			feed.Items = append(feed.Items, Item{
				ID:        e.ID,
				Title:     strings.TrimSpace(e.Title),
				Link:      strings.TrimSpace(link),
				Published: strings.TrimSpace(published),
			})
		}
		return feed, nil

	case "RDF":
		var doc rdfDoc
		if err := xml.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("decode rdf: %w", err)
		}
		feed := &Feed{Title: strings.TrimSpace(doc.Channel.Title)}
		for _, it := range doc.Items {
			id := it.About
			if id == "" {
				id = it.Link
			}
			feed.Items = append(feed.Items, Item{
				ID:        id,
				Title:     strings.TrimSpace(it.Title),
				Link:      strings.TrimSpace(it.Link),
				Published: strings.TrimSpace(it.Date),
			})
		}
		return feed, nil

	default:
		return nil, fmt.Errorf("unrecognized feed format (root element %q)", root)
	}
}

// rootElementName peeks at the first start element so we know whether to
// decode as RSS, Atom, or RSS 1.0/RDF before committing to a struct shape.
// RDF feeds put <item> elements as siblings of <channel> rather than
// nesting them inside it, so they get their own struct shape.
func rootElementName(data []byte) (string, error) {
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", fmt.Errorf("read xml: %w", err)
		}
		if se, ok := tok.(xml.StartElement); ok {
			return se.Name.Local, nil
		}
	}
}
