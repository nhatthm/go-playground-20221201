package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/bool64/ctxd"

	"github.com/nhatthm/go-playground-20221201/internal/crawler"
)

const jsonIndent = "  "

// resultWriter is a function that writes the results of a crawler to a writer.
type resultWriter func(results <-chan crawler.LinkCrawlerResult) ExitCode

// nolint: tagliatelle
type crawlerResult struct {
	PageURL          string  `json:"page_url"`
	NumInternalLinks int     `json:"internal_links_num"`
	NumExternalLinks int     `json:"external_links_num"`
	Success          bool    `json:"success"`
	Error            *string `json:"error"`
}

// bufferedJSONResultWriter creates a new result writer that writes the crawled results to memory and then the output at the end of the process.
//
// In case of error while writing to the output, the error will be logged and the process will stop with exit code CodeErrOutput.
func bufferedJSONResultWriter(out io.Writer, pretty bool, log ctxd.Logger) resultWriter {
	return func(results <-chan crawler.LinkCrawlerResult) (code ExitCode) {
		code = CodeOK
		ctx := context.Background()
		buf := make([]crawlerResult, 0)

		defer func() {
			enc := json.NewEncoder(out)

			if pretty {
				enc.SetIndent("", jsonIndent)
			}

			if err := enc.Encode(buf); err != nil {
				code = CodeErrOutput

				log.Error(ctx, "failed to encode report", "error", err)
			}
		}()

		for r := range results {
			log.Debug(ctx, "received result", "result", r)

			buf = append(buf, toCrawlerResult(r))
		}

		return code
	}
}

// unbufferedJSONResultWriter creates a new result writer that writes the crawled results to output.
//
// In case of error while writing to the output, the error will be printed to the error output and the process will stop with exit code CodeErrOutput.
func unbufferedJSONResultWriter(out, outErr io.Writer, pretty bool) resultWriter {
	return func(results <-chan crawler.LinkCrawlerResult) (code ExitCode) {
		writeErr := func(format string, args ...interface{}) {
			code = CodeErrOutput
			_, _ = fmt.Fprintf(outErr, format, args...)
		}

		buf := new(bytes.Buffer)
		enc := json.NewEncoder(buf)
		join := ""

		newL, startIndent, joinTmpl := "", "", ","

		if pretty {
			newL, startIndent = "\n", jsonIndent
			joinTmpl = ",\n" + startIndent

			enc.SetIndent(jsonIndent, jsonIndent)
		}

		if _, err := fmt.Fprint(out, "[", newL, startIndent); err != nil {
			writeErr("could not write [ to output: %s\n", err)

			return
		}

		defer func() {
			if code != CodeOK {
				return
			}

			if _, err := fmt.Fprint(out, newL, "]\n"); err != nil {
				writeErr("could not write ] to output: %s\n", err)
			}
		}()

		for result := range results {
			buf.Reset()

			if err := enc.Encode(toCrawlerResult(result)); err != nil { // This should not happen.
				writeErr("could not encode %q report: %s", result.Source, err.Error())

				return
			}

			if _, err := fmt.Fprint(out, join, strings.Trim(buf.String(), "\r\n")); err != nil {
				writeErr("could not write %q report: %s", result.Source, err.Error())

				return
			}

			join = joinTmpl
		}

		return CodeOK
	}
}

// toCrawlerResult converts a crawler.LinkCrawlerResult to crawlerResult for output.
func toCrawlerResult(r crawler.LinkCrawlerResult) crawlerResult {
	result := crawlerResult{
		PageURL:          r.Source,
		NumInternalLinks: len(r.InternalLinks),
		NumExternalLinks: len(r.ExternalLinks),
		Success:          r.Error == nil,
	}

	if r.Error != nil {
		err := r.Error.Error()
		result.Error = &err
	}

	return result
}
