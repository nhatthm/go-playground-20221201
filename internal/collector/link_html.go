package collector

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
)

var _ LinkCollector = (*HTMLLinkCollector)(nil)

// HTMLLinkCollector is a collector that collects links from a reader of an HTML document.
//
//    c := NewHTMLLinkCollector()
//    links, err := c.GetLinks(r)
//    if err != nil {
//    	return nil, err
//    }
//
//    fmt.Println(links)
type HTMLLinkCollector struct {
	tagAttributes map[string]string // Key is tag name, Value is attribute name.
}

// GetLinks collects links from a reader of an HTML document.
func (c HTMLLinkCollector) GetLinks(r io.Reader) ([]string, error) {
	z := html.NewTokenizer(r)
	links := make([]string, 0, initialLinksCapacity)

process:
	for {
		switch tt := z.Next(); tt { // nolint: exhaustive // We ignore the other tokens because we focus on the tag attributes.
		case html.ErrorToken:
			if errors.Is(z.Err(), io.EOF) {
				break process
			}

			return nil, fmt.Errorf("could not collect links from html doc: %w", z.Err())

		case html.StartTagToken, html.SelfClosingTagToken:
			tag := z.Token()
			if wantAttr, ok := c.tagAttributes[tag.Data]; ok {
				for _, attr := range tag.Attr {
					if attr.Key == wantAttr {
						// In HTML, \n does not mean new line. Browser will ignore it, so link like "\nhttps://example.org/\npath" will be interpreted
						// as "https://example.org/path".
						links = append(links, strings.ReplaceAll(attr.Val, "\n", ""))

						break
					}
				}
			}
		}
	}

	// Reduce memory allocation. GC will clean up the old links slice.
	result := make([]string, len(links))
	copy(result, links)

	return result, nil
}

// NewHTMLLinkCollector creates a new collector for collecting links from an HTML document.
//
//    c := NewHTMLLinkCollector()
//    links, err := c.GetLinks(r)
//    if err != nil {
//    	return nil, err
//    }
//
//    fmt.Println(links)
func NewHTMLLinkCollector() *HTMLLinkCollector {
	return &HTMLLinkCollector{
		tagAttributes: map[string]string{
			"a": "href",
		},
	}
}
