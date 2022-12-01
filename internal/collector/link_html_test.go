//go:build !testsignal

package collector_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nhatthm/go-playground-20221201/internal/collector"
)

const sampleHTML = "../../resources/fixtures/sample.html"

func TestHTMLLinkCollector_GetLinks_Error(t *testing.T) {
	t.Parallel()

	c := collector.NewHTMLLinkCollector()

	actual, err := c.GetLinks(newErrorReader(errors.New("random error")))

	assert.EqualError(t, err, "could not collect links from html doc: random error")
	assert.Nil(t, actual)
}

func TestHTMLLinkCollector_GetLinks_Success(t *testing.T) {
	t.Parallel()

	f, err := os.Open(filepath.Clean(sampleHTML))
	require.NoError(t, err, "could not open html fixture")

	defer f.Close() // nolint: errcheck,gosec

	c := collector.NewHTMLLinkCollector()

	actual, err := c.GetLinks(f)
	require.NoError(t, err, "could not get links")

	expected := []string{
		"http://google.com",
		"http://www.google.com",
		"https://google.com",
		"https://www.google.com",
		"/",
		"/absolute/path",
		"relative/path",
		"#anchor",
		"?message=hello%20world",
		".",
		"",
		"https://example.org/link-is-broken",
		"javascript:alert('hello')",
		"mailto:john@example.com",
		"http://www.bing.com",
	}

	assert.Equal(t, expected, actual)
}
