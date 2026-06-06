package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("gatling", flag.ContinueOnError)
	fs.SetOutput(stderr)

	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: gatling [options] <url>\n\n")
		fmt.Fprint(stderr, "Options:\n")
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
		fmt.Fprintln(stderr, err)
		return 2
	}

	fmt.Fprintln(stdout, options)
	return 0
}
