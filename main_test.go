package main

import (
	"io/ioutil"
	"net/url"
	"reflect"
	"testing"

	"github.com/hioki-daichi/parallel-download/opt"
)

func TestMain_genFilename(t *testing.T) {
	cases := map[string]struct {
		url      string
		output   string
		expected string
	}{
		"index.html": {url: "http://example.com", output: "", expected: "index.html"},
		"foo.png":    {url: "http://example.com/foo.png", output: "", expected: "foo.png"},
	}

	for n, c := range cases {
		c := c
		t.Run(n, func(t *testing.T) {
			t.Parallel()

			expected := c.expected

			opts := &opt.Options{
				Parallelism: 8,
				Output:      c.output,
				URL:         mustParseRequestURI(t, c.url),
			}

			d := newDownloader(ioutil.Discard, opts)

			actual, err := d.genFilename()
			if err != nil {
				t.Fatalf("err %s", err)
			}
			if actual != expected {
				t.Errorf(`unexpected filename: expected: "%s" actual: "%s"`, expected, actual)
			}
		})
	}
}

func TestMain_genFilename_ErrExist(t *testing.T) {
	t.Parallel()

	expected := errExist

	opts := &opt.Options{
		Parallelism: 8,
		Output:      "",
		URL:         mustParseRequestURI(t, "http://example.com/main_test.go"),
	}

	d := newDownloader(ioutil.Discard, opts)

	_, actual := d.genFilename()
	if actual != expected {
		t.Errorf(`unexpected : expected: "%s" actual: "%s"`, expected, actual)
	}
}

func TestMain_genFilename_HasOutput(t *testing.T) {
	t.Parallel()

	expected := "foo.png"

	opts := &opt.Options{
		Parallelism: 8,
		Output:      expected,
		URL:         nil,
	}

	d := newDownloader(ioutil.Discard, opts)

	actual, err := d.genFilename()
	if err != nil {
		t.Fatalf("err %s", err)
	}
	if actual != expected {
		t.Errorf(`unexpected filename: expected: "%s" actual: "%s"`, expected, actual)
	}
}

func TestMain_generateFormattedRangeHeaders(t *testing.T) {
	cases := map[string]struct {
		contentLength int
		parallelism   int
		expected      []string
	}{
		"contentLength: 5, parallelism: 0": {contentLength: 5, parallelism: 0, expected: []string{"bytes=0-4"}},
		"contentLength: 5, parallelism: 1": {contentLength: 5, parallelism: 1, expected: []string{"bytes=0-4"}},
		"contentLength: 5, parallelism: 2": {contentLength: 5, parallelism: 2, expected: []string{"bytes=0-1", "bytes=2-4"}},
		"contentLength: 5, parallelism: 3": {contentLength: 5, parallelism: 3, expected: []string{"bytes=0-0", "bytes=1-1", "bytes=2-4"}},
		"contentLength: 5, parallelism: 4": {contentLength: 5, parallelism: 4, expected: []string{"bytes=0-0", "bytes=1-1", "bytes=2-2", "bytes=3-4"}},
		"contentLength: 5, parallelism: 5": {contentLength: 5, parallelism: 5, expected: []string{"bytes=0-0", "bytes=1-1", "bytes=2-2", "bytes=3-3", "bytes=4-4"}},
		"contentLength: 5, parallelism: 6": {contentLength: 5, parallelism: 6, expected: []string{"bytes=0-0", "bytes=1-1", "bytes=2-2", "bytes=3-3", "bytes=4-4"}},
	}

	for n, c := range cases {
		c := c
		t.Run(n, func(t *testing.T) {
			t.Parallel()

			contentLength := c.contentLength
			parallelism := c.parallelism
			expected := c.expected

			actual, err := generateFormattedRangeHeaders(contentLength, parallelism)
			if err != nil {
				t.Fatalf("err %s", err)
			}
			if !reflect.DeepEqual(actual, expected) {
				t.Errorf(`expected="%s" actual="%s"`, expected, actual)
			}
		})
	}
}

// func newTestServer(t *testing.T) (*httptest.Server, func()) {
// 	t.Helper()
//
// 	ts := httptest.NewServer(http.HandlerFunc(
// 		func(w http.ResponseWriter, r *http.Request) {
// 			w.Header().Set("Accept-Ranges", "bytes")
// 			// body, err := genBody(t, r)
//
// 			body, err := func(t *testing.T, req *http.Request) (string, error) {
// 				b, err := ioutil.ReadFile("./testdata/foo.png")
// 				if err != nil {
// 					t.Fatalf("err %s", err)
// 				}
// 				contents := string(b)
//
// 				// e.g. "bytes=0-99"
// 				rangeHdr := req.Header.Get("Range")
// 				if rangeHdr == "" {
// 					return contents, nil
// 				}
//
// 				// e.g. []string{"bytes", "0-99"}
// 				eqlSplitVals := strings.Split(rangeHdr, "=")
// 				if eqlSplitVals[0] != "bytes" {
// 					return "", errors.New(`only "bytes" is accepted`)
// 				}
//
// 				// e.g. []string{"0", "99"}
// 				c := strings.Split(eqlSplitVals[1], "-")
//
// 				// e.g. 0
// 				start, err := strconv.Atoi(c[0])
// 				if err != nil {
// 					return "", err
// 				}
//
// 				// e.g. 99
// 				end, err := strconv.Atoi(c[1])
// 				if err != nil {
// 					return "", err
// 				}
//
// 				// e.g. "Range: bytes=1-0"
// 				if start > end {
// 					return "", errors.New("invalid range")
// 				}
//
// 				l := len(contents)
// 				if end > l {
// 					end = l
// 				}
//
// 				return contents[start : end+1], nil
// 			}(t, r)
//
// 			if err != nil {
// 				w.WriteHeader(http.StatusBadRequest)
// 				fmt.Fprint(w, err.Error())
// 				return
// 			}
// 			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
// 			fmt.Fprint(w, body)
// 			return
// 		},
// 	))
//
// 	return ts, func() { ts.Close() }
// }

func mustParseRequestURI(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.ParseRequestURI(s)
	if err != nil {
		t.Fatalf("err %s", err)
	}
	return u
}
