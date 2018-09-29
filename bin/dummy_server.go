//usr/bin/env go run $0 $@ ; exit

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	flg := flag.NewFlagSet("test", flag.ExitOnError)

	port := flg.Int("port", 8080, "Port on which the dummy server listens.")
	failureRate := flg.Int("failure-rate", 0, "Probability to return InternalServerError.")
	maxDelay := flg.Duration("max-delay", time.Second, "Maximum time delay randomly applied from receiving a request until returning a response.")

	flg.Parse(os.Args[1:])

	fmt.Printf("=> starting with a failure rate of %d%% on http://localhost:%d\n", *failureRate, *port)
	fmt.Println(`================================================================================
THIS IS A DUMMY SERVER THAT CAN PARTIALLY RETURN IMAGE DATA !!
================================================================================`)
	fmt.Printf("Usage:\n")
	flg.PrintDefaults()
	fmt.Printf("Endpoint:\n  GET /foo.png # Get a gopher image\n")

	contents := func() string {
		b, err := ioutil.ReadFile("./downloading/testdata/foo.png")
		if err != nil {
			panic(err)
		}
		return string(b)
	}()

	handler := func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "HEAD" {
			time.Sleep(time.Duration(rand.Intn(int(*maxDelay))))
		}

		w.Header().Set("Accept-Ranges", "bytes")

		var body string
		var statusCode int
		if req.Method == "GET" && rand.Intn(100) < *failureRate {
			body = "Internal Server Error"
			statusCode = http.StatusInternalServerError
		} else {
			body = func(req *http.Request) string {
				rangeHeader := req.Header.Get("Range") // e.g. "bytes=0-99"
				if rangeHeader == "" {
					return contents
				}
				c := strings.Split(strings.Split(rangeHeader, "=")[1], "-")
				min, _ := strconv.Atoi(c[0])
				max, _ := strconv.Atoi(c[1])
				return contents[min : max+1]
			}(req)
			statusCode = http.StatusPartialContent
		}

		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
		w.WriteHeader(statusCode)
		fmt.Fprint(w, body)

		log.Printf("%s %s %d %s\n", req.Method, req.RequestURI, statusCode, req.Header.Get("Range"))
	}

	http.HandleFunc("/foo.png", handler)
	http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
}
