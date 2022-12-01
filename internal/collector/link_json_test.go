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

const sampleJSON = "../../resources/fixtures/sample.json"

func TestJSONLinkCollector_GetLinks_Error(t *testing.T) {
	t.Parallel()

	c := collector.NewJSONLinkCollector()

	actual, err := c.GetLinks(newErrorReader(errors.New("random error")))

	assert.EqualError(t, err, "could not collect links from json doc: random error")
	assert.Nil(t, actual)
}

func TestJSONLinkCollector_GetLinks_Success(t *testing.T) {
	t.Parallel()

	f, err := os.Open(filepath.Clean(sampleJSON))
	require.NoError(t, err, "could not open html fixture")

	defer func() {
		_ = f.Close() // nolint: errcheck
	}()

	c := collector.NewJSONLinkCollector()

	actual, err := c.GetLinks(f)
	require.NoError(t, err, "could not get links")

	expected := []string{
		"http://google.com",
		"http://www.google.com",
		"https://google.com",
		"https://www.google.com",
		"http://www.bing.com",
		"https://google.com/?q=hello+world",
		"https://bing.com/?q=link+in+key",
		"https://google.com/?q=array+element+1",
		"https://bing.com/?q=array+element+2",
		"https://google.com/?q=long+text+1",
		"https://bing.com/?q=long+text+2",
		"https://ran-dom.com/",
	}

	assert.Equal(t, expected, actual)
}
