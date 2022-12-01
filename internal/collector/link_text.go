package collector

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
)

var _ LinkCollector = (*TextLinkCollector)(nil)

// httpLinkRegexp is the regex used to extract links from a string.
// Ref: https://mathiasbynens.be/demo/url-regex
var httpLinkRegexp = regexp.MustCompile(`(https?)://(-\.)?([^\s/?.#]+\.?)+(/\S*)?`)

// TextLinkCollector is a collector that collects links from a reader of a text document.
//
//    c := NewTextLinkCollector()
//    links, err := c.GetLinks(r)
//    if err != nil {
//    	return nil, err
//    }
//
//    fmt.Println(links)
type TextLinkCollector struct{}

// GetLinks collects links from a reader of an HTML document.
func (t TextLinkCollector) GetLinks(r io.Reader) ([]string, error) {
	s := bufio.NewScanner(r)
	links := make([]string, 0, initialLinksCapacity)

	for s.Scan() {
		links = append(links, httpLinkRegexp.FindAllString(s.Text(), -1)...)
	}

	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("could not collect links from text doc: %w", err)
	}

	// Reduce memory allocation. GC will clean up the old links slice.
	result := make([]string, len(links))
	copy(result, links)

	return result, nil
}

// NewTextLinkCollector creates a new collector for collecting links from a text document.
//
//    c := NewTextLinkCollector()
//    links, err := c.GetLinks(r)
//    if err != nil {
//    	return nil, err
//    }
//
//    fmt.Println(links)
func NewTextLinkCollector() *TextLinkCollector {
	return &TextLinkCollector{}
}
