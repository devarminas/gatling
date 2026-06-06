package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func main() {
	client := http.Client{Timeout: 30 * time.Second}
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, &client))
}

func run(args []string, stdout, stderr io.Writer, client *http.Client) int {
	fs := flag.NewFlagSet("gatling", flag.ContinueOnError)
	fs.SetOutput(stderr)

	fs.Usage = func() {
		_, _ = fmt.Fprint(stderr, "Usage: gatling [options] <url>\n\n")
		_, _ = fmt.Fprint(stderr, "Options:\n")
		fs.PrintDefaults()
	}

	requests := fs.Int("n", 10, "total number of requests to send")
	concurrency := fs.Int("c", 1, "how many requests run at the same time")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}

	url := fs.Arg(0)
	options := config{requests: *requests, concurrency: *concurrency, url: url}
	if err := options.validate(); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}

	results := runSequential(client, options)
	if _, err := fmt.Fprintln(stdout, results); err != nil {
		_, _ = fmt.Fprintf(stderr, "write output: %v\n", err)
		return 1
	}
	return 0
}
