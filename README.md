# Playground 20221201

<!--[![Build Status](https://github.com/nhatthm/go-playground-20221201/actions/workflows/test.yaml/badge.svg?branch=dev)](https://github.com/nhatthm/go-playground-20221201/actions/workflows/test.yaml)
[![codecov](https://codecov.io/gh/nhatthm/go-playground-20221201/branch/dev/graph/badge.svg?token=pQqEb2AUcE)](https://codecov.io/gh/nhatthm/go-playground-20221201)-->

An awesome crawler that counts for internal and external links from given URLs.

## Table of Contents

- [Getting started](#getting-started)
    - [Prerequisites](#prerequisites)
    - [Development](#development)
        - [Dependencies](#dependencies)
        - [Test](#test)
            - [Unit Test](#unit-test)
            - [Signal Test](#signal-test)
        - [Build](#build)
        - [Run the tool locally](#run-the-tool-locally)
    - [Code Conventions](#code-conventions)
    - [Continuous Integration](#continuous-integration)
- [Usage](#usage)
- [Examples](#examples)
- [Output](#output)
- [Features](#features)
    - [Multiple sources supported](#multiple-sources-supported)
    - [Multiple data types supported](#multiple-data-types-supported)
    - [Adaptive Output](#adaptive-output)
    - [Streaming Output](#streaming-output)
- [Project Structure](#project-structure)
- [Design](#design)
- [To be or not to be - Internal vs External](#to-be-or-not-to-be---internal-vs-external)
- [Limits and Future Enhancements](#limits-and-future-enhancements)
    - [Links without `scheme` or `hostname` in `text/plain` or `application/json`](#links-without-scheme-or-hostname-in-textplain-or-applicationjson)
    - [Collect more links than `a[href]` in `text/html` document](#collect-more-links-than-ahref-in-texthtml-document)
    - [More convenient Adaptive Output](#more-convenient-adaptive-output)
    - [Support more media types](#support-more-media-types)
    - [Support more encoding](#support-more-encoding)
    - [Split link extraction out of `Collector`](#split-link-extraction-out-of-collector)
    - [Dynamic maximum number of workers](#dynamic-maximum-number-of-workers)
- [GDPR](#gdpr)
- [Resources](#resources)

## Getting started

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

### Prerequisites

- `Go >= 1.17`
- [`golangci-lint v1.46.2`](https://github.com/golangci/golangci-lint/releases/tag/v1.46.2)
- `git`
- `Makefile`

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

### Development

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

#### Dependencies

Beside built-in dependencies, this project needs some external dependencies for its operation.

| Dependency                    | Reason                                                                                                   |
|-------------------------------|----------------------------------------------------------------------------------------------------------|
| `golang.org/x/net`            | Need the `html` subpackage for parsing HTML document                                                     |
| `github.com/bool64/ctxd`      | Need for contextualized, structured, and level logging                                                   |
| `github.com/bool64/zapctxd`   | Need for using Uber's `zap` logger with `bool64/ctxd`                                                    |
| `go.uber.org/zap`             | Need for structured, and leveled logging                                                                 |
| `github.com/nhatthm/httpmock` | Need for testing with HTTP server. This is a powerful wrapper on top of the built-in `net/http/httptest` |
| `github.com/stretchr/testify` | Need for test assertions                                                                                 |

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

#### Test

The tool is tested with Unit Test and Signal Test by running `make test`.

- The Unit Test is run with [`nhatthm/httpmock`](https://github.com/nhatthm/httpmock) to ensure that the tool works with real URLs, follows the protocol, and
  supports all the needs.
- In order to test `SIGINT`/`SIGTERM`, the test process has to be interrupted by calling `syscall.Kill(syscall.Getpid(), syscall.SIGINT)` inside the test case.
  That affects all other running in-parallel test cases. Therefore, there is the Signal Test with a dedicated build tag `testsignal` to enable that test case.

All the tests are run with `t.Parallel()` and [Race Detector](https://go.dev/blog/race-detector) (the `-race` flag) to prevent race condition.

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

##### Unit Test

Run `make test-unit`.

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

##### Signal Test

Run `make test-signal`.

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

#### Build

Run `make build` to build the application, the `cli` binary will be generated in the `out` directory.

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

#### Run the tool locally

[Build the project](#build) then run `out/cli`. Read more about its [Usage](#usage) and [Examples](#examples).

You could also run it with your IDE by pointing to the `cmd/cli` directory where the `main()` function is, or with `go run cmd/cli/main.go`.

```
$ go run cmd/cli/main.go -p 1 google.com
[
  {
    "page_url": "goog.com",
    "internal_links_num": 5,
    "external_links_num": 17,
    "success": true,
    "error": null
  }
]
```

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

### Code Conventions

The project follow the community best practices for code formatting and style. There is a `lint` pipeline to check for code style for every pull requests and
the main branch.

On local, you can run `make lint` to check for code style, it requires `golangci-lint` to be installed.

The linter configuration is in `.golangci.yml`. There is also `.editorconfig` file to help the IDE to format the code.

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

### Continuous Integration

This project uses [GitHub Actions](https://docs.github.com/en/actions), you can find the pipelines in `.github/workflows/` folder.

However, this is a private repository in a personal account with a free budget so keep in mind that you may not be able to run the CI pipeline when it hits the
limit.

There are 2 pipelines that run for every pull requests and the main branch:

- `lint`: Run `make lint` to lint the codebase.
- `test`: Run `make test` to run the unit test and the signal test.

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

## Usage

[Build the project](#build) then run `out/cli [options] [link1 link2 ... linkN]`

```
Usage:
  cli [options] [link1 link2 ... linkN]

Options:
  -f, --file PATH/TO/FILE
                    Path to the input file that contains a list of urls,
                    separated by '\n'.
                    This option is used if no links are provided.
  -p, --parallel NUM
                    Number of workers for crawling. Default to 10.
  -t, --timeout TIMEOUT
                    Timeout for requesting an url, in the form "72h3m0.5s".
                    Default to 30s.
  --no-pretty       Disable pretty output.
  -v, --verbose     Print out the error log messages.
  -vv               Print out the all log messages.
  -h, --help        Print out the help message.
```

- The `-p, --parallel` is optional, default to `10`. Only an integer between `1` and `24` is accepted.
- The `-t, --timeout` is optional, default to `30s`. See [Time Duration format](https://golang.org/pkg/time/#ParseDuration) for the timeout format.
- All URLs can be with or without `scheme` or `www` prefix, but must have a `hostname`. If the `scheme` is missing, default to `https`.
- The tool will check the links in the arguments first.
    - If there is none, it will check for the input file.
    - If there is no input file, it will check for piped `stdin`.
    - If there is no other option, it will yield an error.

Return Code:

| Code | Name                            | Description                                                                     |
|:----:|:--------------------------------|:--------------------------------------------------------------------------------|
| `0`  | `CodeOK`                        | The tool exited with success                                                    |
| `1`  | `CodeErrOperationCanceled`      | The tool has been terminated by `SIGINT` or `SIGTERM` and operation is canceled |
| `2`  | `CodeErrNoInputSource`          | The tool has no input source                                                    |
| `3`  | `CodeErrOpenInputSource`        | The tool couldn't open input file given by `-f, --file`                         |
| `4`  | `CodeErrUnsupportedInputSource` | The tool couldn't use the input source                                          |
| `5`  | `CodeErrBadArgs`                | The provided arguments are invalid                                              |
| `6`  | `CodeErrOutput`                 | The tool couldn't write to the output stream                                    |

Examples:

- Crawl all the urls in `path/to/file.txt`<br/>
  `out/cli -p 24 -i path/to/file.txt`
- Crawl all the urls in arguments<br/>
  `out/cli -p 10 google.com facebook.com`
- Crawl all the urls piped in `stdin`<br/>
  `echo $'google.com\nfacebook.com' | out/cli -p 10`
- Crawl with timeout<br/>
  `out/cli -t 10s google.com`
- Crawl with debug mode<br/>
  `out/cli -vv google.com`

<p align="center">
  <img src="https://user-images.githubusercontent.com/1154587/174740963-9442e82c-dfd3-46be-912b-dfff2d72f0a9.png" alt="cli" width="75%"><br/>
    <sub><i>Screenshot of the application running with debug mode</i></sub>
</p>

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

## Examples

There are 2 built-in example commands in the `Makefile`:

- `make example`: runs the crawler with urls in the arguments.
- `make example-file`: runs the crawler with `-f` option.

The data source for the 2 commands is
in [`resources/fixtures/sources.txt`](https://github.com/nhatthm/go-playground-20221201/blob/dev/resources/fixtures/sources.txt). This file contains a list of
URLs, one of them has `404 Not Found`.

There are 2 options to run with those commands above:

|   Option    |   Type   | Description                                                                                            |
|:-----------:|:--------:|:-------------------------------------------------------------------------------------------------------|
| `PARALLEL`  |  `int`   | Number of workers, will be passed to `-p, --parallel` option                                           |
| `LOG_LEVEL` | `string` | Log level, will be converted to `-v, --verbose` option.<br/>Use `error` for `-v` and `debug` for `-vv` |

For example:

- `make example-file PARALLEL=10 LOG_LEVEL=debug`

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

## Output

The tool will output a JSON array of result objects to `stdout` in this structure:

|        Field         |   Type   | Nullable | Description                                                                                |
|:--------------------:|:--------:|:--------:|:-------------------------------------------------------------------------------------------|
|      `page_url`      | `string` |    No    | The original url that provided by the input source                                         |
| `internal_links_num` |  `int`   |    No    | The number of internal links in the response                                               |
| `external_links_num` |  `int`   |    No    | The number of internal links in the response                                               |
|      `success`       |  `bool`  |    No    | Whether the request is successful. It is `true` when `error` is `null`. Otherwise, `false` |
|       `error`        | `string` |   Yes    | In case of error, the field is a string of error message. Otherwise, it's `null`           |

For example:

```json
[
    {
        "page_url": "samsung.com",
        "internal_links_num": 520,
        "external_links_num": 27,
        "success": true,
        "error": null
    }
]
```

And the log messages will be printed to `stderr` in the following format:

```
{timestamp}\t{level}\t{caller}\t{message}\t{context data}
```

For example:

```
2022-06-21T14:38:14.177+0200	DEBUG	crawler/link_http.go:109	started crawling	{"crawler.http.worker_id": 2, "crawler.http.source": "google.com"}
```

_Note:_ if you don't want prettified output, use `--no-pretty`.

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

## Features

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

### Multiple sources supported

The tool could take the links from the arguments, or from the input file, or from the piped `stdin`. The priority is given from left to right.

This flexibility is useful when you want to integrate the tool with other tools, use it in a CI pipeline, or use it in a script.

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

### Multiple data types supported

The tool collects the links according to the `Content-Type` of the response. However, it lacks support of compression (`Content-Encoding`). If the response does
not provide the `Content-Type`, or the value is `application/octet-stream`, the tool will try to detect the real media type by
using [`http.DetectContentType()`](https://pkg.go.dev/net/http#DetectContentType).

Media Types:

|         Media Type         | Supported | Note                                                                                                 |
|:--------------------------:|:---------:|:-----------------------------------------------------------------------------------------------------|
|        `text/plain`        |    Yes    | The tool reads only links that start with `http://` or `https://`                                    |
|        `text/html`         |    Yes    | The tool reads only `href` attr of `a` tag                                                           |
|     `application/json`     |    Yes    | The tool reads only links that start with `http://` or `https://` in the keys or string values       |
|       `text/x-json`        |    Yes    | Same as `application/json`                                                                           |
| `application/octet-stream` |  Depends  | Depends on the result of the detection. If it's still `application/octet-stream`, it's not supported |
|     `application/xml`      |    No     ||
|     `application/pdf`      |    No     ||
|          `Others`          |    No     ||

Content Encoding:

| Compressor | Supported | Note                         |
|:----------:|:---------:|:-----------------------------|
|   `gzip`   |    Yes    | Built-in support from Golang |
|    `br`    |    No     ||
|  `bzip2`   |    No     ||
| `deflate`  |    No     ||
|    `xz`    |    No     ||

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

### Adaptive Output

When you run with `-v, -vv, --verbose` option, the tool will output the log messages as well as the result objects. However, for human users, both stream will
be displayed in the same terminal. Which means it's extremely heard to read.

In order to solve that problem, in case of debugging, the tool will keep the results in memory and flush that to `stdout` at the end when all the work is done.

Otherwise, the tool will output the results to `stdout` as soon as it is available.

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

### Streaming Output

Thanks to the unbuffered output, the tool could easily stream the results to the user, or a file, or a pipe as soon as it is available.

For example: streaming a list of sources to the tool, then filter for urls that have more than 50 internal links.

```
$ cat resources/fixtures/sources.txt | out/cli -p 20 -t 1m | jq -r --stream 'fromstream(1|truncate_stream(inputs)) | select(.internal_links_num > 50) | "\(.page_url)"'
https://www.postgresql.org/
weather.com
shopify.com
msn.com
amazon.com
samsung.com
mysql.com
```

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

## Project Structure

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

### `cmd/cli`

The entry point of the `cli` application. It doesn't contain any logic about the tool, or how to run it.

The main purpose of this package is to provide the command line interface of the tool. It will parse the arguments, and then call the `Run()` function
in `internal/app/cli` package.

This is also good for testing because we can't test `main()` function.

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

### `internal/app/cli`

The package contains the actual logic of the tool. It will take the arguments, initiate all the needed services and then run them.

The configuration is straightforward

```go
package cli

type Config struct {
	OutWriter io.Writer
	ErrWriter io.Writer

	NumWorkers     int
	Timeout        time.Duration
	PrettyOutput   bool
	VerbosityLevel VerbosityLevel
}
```

|      Config      | Description                                                  |
|:----------------:|:-------------------------------------------------------------|
|   `OutWriter`    | The stream that will receive the results                     |
|   `ErrWriter`    | The stream that will receive all the log messages and errors |
|   `NumWorkers`   | The number of workers that the crawler could run             |
|    `Timeout`     | The timeout of the http client of the crawler                |
|  `PrettyOuptut`  | Disable JSON prettifier                                      |
| `VerbosityLevel` | The verbosity level of the tool                              |

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

### `internal/collector`

The `Collector` interface that reads the stream and extract the links.

```go
package collector

// LinkCollector is a collector that collects links from a reader.
type LinkCollector interface {
	GetLinks(r io.Reader) ([]string, error)
}
```

Current collectors:

|      Collector      | Description                      |
|:-------------------:|:---------------------------------|
| `HTMLLinkCollector` | Collect links from HTML document |
| `TextLinkCollector` | Collect links from text document |
| `JSONLinkCollector` | Collect links from JSON document |

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

### `internal/crawler`

The `Crawler` interface that crawl for internal and externals from different urls.

There are some options to set up the `HTTPLinkCrawler`

| Option                                                                         | Description                                                |
|:-------------------------------------------------------------------------------|:-----------------------------------------------------------|
| `WithLinkCollectors(collectors map[string]collector.LinkCollector)`            | Set the list of supported media types and their collectors |
| `WithLinkCollector(collector collector.LinkCollector, contentTypes ...string)` | Set the collector for some specific media types            |
| `WithNumWorkers(numWorkers int)`                                               | Set the number of workers                                  |
| `WithClientTimeout(d time.Duration)`                                           | Set the timeout of the http client                         |
| `WithLogger(l ctxd.Logger)`                                                    | Set the logger                                             |

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

### `internal/logger`

Set up the `zapctxd.Logger` for the project.

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

### `resources/fixtures`

Fixtures for the tests and examples.

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

## Design

<p align="center">
  <img src="https://user-images.githubusercontent.com/1154587/174855520-5edc1a33-6d2b-44c3-a407-618e68d68d87.png" alt="architecture" width="80%">
</p>

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

## To be or not to be - Internal vs External

- With html documents, the `Collector` will collect all the values in the `href` attribute of the `a` tag,
- With plain text and JSON documents, the `Collector` will get only links that start with `http` or `https`

Then the crawler will sort them with the following logic:

- If the link has a `scheme` that is different than `http` or `https`, e.g `mailto`, `tel`, `javascript`, etc. it will be discarded.
- If the link has a `host` that is different than the source link, it will be sorted as External.
- The link is now sorted as Internal.

For example: source is `example.com/category/page`

| Example                       | Result    | Resolved as                                      |
|:------------------------------|:----------|:-------------------------------------------------|
| `mailto:admin@example.org`    | Discarded ||
| `tel:112`                     | Discarded ||
| `javascript:alert("hello")`   | Discarded ||
| `https://google.com`          | External  | `https://google.com`                             |
| `http://example.com`          | Internal  | `http://example.com`                             |
| `https://example.com`         | Internal  | `https://example.com`                            |
| `https://sub.example.com`     | External  | `https://sub.example.com`                        |
| `.`                           | Internal  | `https://example.com/category/page`              |
| `./`                          | Internal  | `https://example.com/category/page`              |
| `path/to/something`           | Internal  | `https://example.com/category/path/to/something` |
| `/absolute/path/to/something` | Internal  | `https://example.com/absolute/path/to/something` |
| `#anchor`                     | Internal  | `https://example.com/category/page#anchor`       |

Read more: [`URL.ResolveReference()`](https://pkg.go.dev/net/url#URL.ResolveReference)

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

## Limits and Future Enhancements

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

### Links without `scheme` or `hostname` in `text/plain` or `application/json`

Unfortunately, due to time constraints, the tool cannot detect the links without `scheme` or `hostname`. It needs more time to form a proper solution.

For the JSON, it seems that the tool could guess by reading the key if it contains `url`, `uri`, `location`, or `href`.

For the plain text, it is harder and goes beyond the scope of this tool. It properly needs a text processing library to do that.

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

### Collect more links than `a[href]` in `text/html` document

To keep the project simple, the tool only reads the `href` attribute of the `a` tag. However, it is very easy to read different attributes of different tags.
The list of supported tag+attribute pairs
is [hardcoded in the constructor](https://github.com/nhatthm/go-playground-20221201/blob/dev/internal/collector/link_html.go#L74-L80).

The tool could also read the text node, but it isn't implemented yet. It should be easy (to do) because the tool tokenizes the html doc, it just needs another
case for the `TextToken` in [this switch](https://github.com/nhatthm/go-playground-20221201/blob/dev/internal/collector/link_html.go#L34).

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

### More convenient Adaptive Output

It is great that the tool has unbuffered and buffered output. However, the buffered output helps only human users but not machines. Because we could redirect
`stderr` to a file while piping the output, it doesn't matter to machines. For example _(silly `cat`, just for demonstration)_:

```
$ out/cli google.com 2>error.txt | cat
```

Therefore, the tool should provide a way to decide whether to use the buffered output or not.

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

### Support more media types

There are still a lot more document media types, such as Word, Excel, PDF, etc. The tool could support them.

The development will take time, but it isn't hard to do so with the current design. We just need to implement the new collectors in the `internal/collector`
package, find a parser library that could parse or tokenize the document.

However, it will go back to the question about text processing
of [links without `scheme` or `hostname`](#links-without-scheme-or-hostname-in-textplain-or-applicationjson).

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

### Support more encoding

This is a valid enhancement. However, it depends on the share (usage), or the popularity of the compressor. `gzip` seems to be leading the market.

According to this [blog post](https://www.ctrl.blog/entry/http-deflate-compression.html) _(unverified source)_

> If found that only 0,015 % of servers returned an HTTP Deflate encoded response. Roughly 30 % of responses used Gzip, and about 10 % used the newer Brotli
> compression format.

It would make sense to support `br` but not `deflate`.

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

### Support more output format.

The tool could support more output formats, one valuable option is `csv`. However, there is a workaround with `jq`, for example:

```
$ out/cli google.com bing.com samsung.com/not-found | jq -r '.[] | "\(.page_url),\(.internal_links_num),\(.external_links_num),\(.success),\(.error//"")"'
google.com,5,17,true,
bing.com,22,13,true,
samsung.com/not-found,0,0,false,unexpected status code: 404
```

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

### Split link extraction out of `Collector`

Currently, every `Collector` has its own parsing and links extraction logic. When the requirement grows, like supporting more media types. There would be a need
to reused or shared those logics.

For example, the HTML collector, it's easy to extract the link from any attributes. However, extracting links without the `scheme` or `hostname` from the text
node isn't easy. It's the same problem as processing plain text or any other documents (e.g. Word, Excel, PDF, etc.). Therefore, the HTML collector would need
two different extractors, and the other collectors could reuse the same extractor of the text node.

In that case, it would make sense to split the link extraction out of `Collector` and introduce a new `Extractor` type.

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

### Dynamic maximum number of workers

The maximum number of workers is hardcoded to `24` in the application. Although the intention (avoiding resource saturation) is good, there is no scientific
reason why that is a good number. It's actually randomly chosen, `42` (
aka [The answer to life, the universe, and everything](https://news.mit.edu/2019/answer-life-universe-and-everything-sum-three-cubes-mathematics-0910)) would
have been a better choice. Coincidentally, `24` is the reverse of `42`.

What we could do here, perhaps:

- Change the number, or log some metrics. Monitor the outcome.
- Find out the relations between the physical cpu resource, memory resource, and the monitoring result.
- Balance them.
- Come up with a formula that could be calculated at runtime.

We could also [write some benchmarks](https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go), that would be a valuable input for the investigation.

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

## GDPR

- The tool does NOT store or send any personal, sensitive, and tracking data to any third party.
- There is NO sending and reading cookies.
- There is NO integration with any third party data storage.
- There is NO tracking integration at any level.

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)

## Resources

- [How Many Links Per Page Is Too Many?](https://moz.com/blog/how-many-links-is-too-many)
- [In search of the perfect URL validation regex](https://mathiasbynens.be/demo/url-regex)
- [Does the web still need HTTP Deflate?](https://www.ctrl.blog/entry/http-deflate-compression.html)
- [How to write benchmarks in Go](https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go)
- [GitHub Actions](https://go.dev/blog/race-detector)
- Golang's [`Race Detector`](https://go.dev/blog/race-detector)
- Golang's [`http.DetectContentType()`](https://pkg.go.dev/net/http#DetectContentType)
- Golang's [`Time Duration format`](https://golang.org/pkg/time/#ParseDuration)
- Golang's [`URL.ResolveReference()`](https://pkg.go.dev/net/url#URL.ResolveReference)

[<sub><sup>[table of contents]</sup></sub>](#table-of-contents)
