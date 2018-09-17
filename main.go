package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"

	"golang.org/x/sync/errgroup"
)

var (
	errExist = errors.New("file already exists")
)

func main() {
	url := os.Args[1]
	err := newDownloader(os.Stdout, url).download()
	if err != nil {
		log.Fatal(err)
	}
}

type downloader struct {
	outStream   io.Writer
	url         string
	parallelism int
}

func newDownloader(w io.Writer, url string) *downloader {
	return &downloader{outStream: w, url: url, parallelism: 8} // TODO: Use flags instead of hard-coded 8
}

func (d *downloader) download() error {
	_, filename := path.Split(d.url)
	_, err := os.Stat(filename)
	if !os.IsNotExist(err) {
		return errExist
	}

	resp, err := http.Head(d.url)
	if err != nil {
		return err
	}

	rangeStrings, err := toRangeStrings(int(resp.ContentLength), d.parallelism)
	if err != nil {
		return err
	}

	responses := map[int]*http.Response{}

	eg := errgroup.Group{}
	for i, rangeString := range rangeStrings {
		i := i
		rangeString := rangeString
		eg.Go(func() error {
			client := &http.Client{Timeout: 0}

			req, err := http.NewRequest("GET", d.url, nil)
			if err != nil {
				return err
			}

			req.Header.Set("Range", rangeString)

			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			fmt.Fprintf(d.outStream, "i: %d, ContentLength: %d, Range: %s\n", i, resp.ContentLength, rangeString)
			responses[i] = resp
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	fp, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	for i := 0; i < len(responses); i++ {
		resp := responses[i]
		_, err := io.Copy(fp, resp.Body)
		if err != nil {
			os.Remove(filename)
			return err
		}
	}

	fmt.Fprintf(d.outStream, "Downloaded: %q\n", d.url)

	return nil
}

func toRangeStrings(contentLength int, parallelism int) ([]string, error) {
	rangeStructs := make([]rangeStruct, 0)

	if parallelism == 0 {
		parallelism = 1
	}

	if contentLength < parallelism {
		parallelism = contentLength
	}

	length := contentLength / parallelism

	i := 0
	for n := parallelism; n > 0; n-- {
		first := i
		i += length
		last := i - 1
		rangeStructs = append(rangeStructs, rangeStruct{first: first, last: last})
	}

	if rem := contentLength % parallelism; rem != 0 {
		rangeStructs[len(rangeStructs)-1].last += rem
	}

	rangeStrings := make([]string, 0)

	for _, rangeStruct := range rangeStructs {
		rangeStrings = append(rangeStrings, fmt.Sprintf("bytes=%d-%d", rangeStruct.first, rangeStruct.last))
	}

	return rangeStrings, nil
}

type rangeStruct struct {
	first int
	last  int
}
