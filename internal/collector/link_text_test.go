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

const sampleText = "../../resources/fixtures/sample.txt"

func TestTextLinkCollector_GetLinks_Error(t *testing.T) {
	t.Parallel()

	c := collector.NewTextLinkCollector()

	actual, err := c.GetLinks(newErrorReader(errors.New("random error")))

	assert.EqualError(t, err, "could not collect links from text doc: random error")
	assert.Nil(t, actual)
}

func TestTextLinkCollector_GetLinks_Success(t *testing.T) {
	t.Parallel()

	f, err := os.Open(filepath.Clean(sampleText))
	require.NoError(t, err, "could not open html fixture")

	defer f.Close() // nolint: errcheck,gosec

	c := collector.NewTextLinkCollector()

	actual, err := c.GetLinks(f)
	require.NoError(t, err, "could not get links")

	expected := []string{
		"http://google.com",
		"http://www.google.com",
		"https://google.com",
		"https://www.google.com",
		"http://www.bing.com",
		"https://google.com/?q=long+text+1",
		"https://bing.com/?q=long+text+2",
		"http://127.0.0.1:8888/path",
		"https://ran-dom.com/",
	}

	assert.Equal(t, expected, actual)
}
