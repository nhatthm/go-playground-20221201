package collector

import "io"

// initialLinksCapacity is the initial capacity of the links slice, it does not mean this is the maximum capacity.
// It is just not recommended to have more than 100 links in a document due to SEO (Page Ranking) reason.
// Ref: https://moz.com/blog/how-many-links-is-too-many
const initialLinksCapacity = 100

// LinkCollector is a collector that collects links from a reader.
type LinkCollector interface {
	GetLinks(r io.Reader) ([]string, error)
}
