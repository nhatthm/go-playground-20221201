package collector

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

var _ LinkCollector = (*JSONLinkCollector)(nil)

// JSONLinkCollector is a collector that collects links from a reader of a text document.
//
//    c := NewJSONLinkCollector()
//    links, err := c.GetLinks(r)
//    if err != nil {
//    	return nil, err
//    }
//
//    fmt.Println(links)
type JSONLinkCollector struct{}

// GetLinks collects links from a reader of an HTML document.
func (t JSONLinkCollector) GetLinks(r io.Reader) ([]string, error) {
	dec := json.NewDecoder(r)
	links := make([]string, 0, initialLinksCapacity)

	for {
		token, err := dec.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return nil, fmt.Errorf("could not collect links from json doc: %w", err)
		}

		if s, ok := token.(string); ok {
			links = append(links, httpLinkRegexp.FindAllString(s, -1)...)
		}
	}

	// Reduce memory allocation. GC will clean up the old links slice.
	result := make([]string, len(links))
	copy(result, links)

	return result, nil
}

// NewJSONLinkCollector creates a new collector for collecting links from a text document.
//
//    c := NewJSONLinkCollector()
//    links, err := c.GetLinks(r)
//    if err != nil {
//    	return nil, err
//    }
//
//    fmt.Println(links)
func NewJSONLinkCollector() *JSONLinkCollector {
	return &JSONLinkCollector{}
}
