package main

import (
	"io"
	"net/http"
	"sync"
	"time"
)

type result struct {
	statusCode int
	latency    time.Duration
	err        error
}

func runRequest(client *http.Client, cfg config) result {
	start := time.Now()
	resp, err := client.Get(cfg.url)
	if err != nil {
		return result{
			latency: time.Since(start),
			err:     err,
		}
	}

	_, readErr := io.Copy(io.Discard, resp.Body)
	closeErr := resp.Body.Close()
	latency := time.Since(start)

	if readErr != nil {
		return result{
			latency:    latency,
			statusCode: resp.StatusCode,
			err:        readErr,
		}
	}

	if closeErr != nil {
		return result{
			latency:    latency,
			statusCode: resp.StatusCode,
			err:        closeErr,
		}
	}

	return result{
		latency:    latency,
		statusCode: resp.StatusCode,
	}
}

func runConcurrent(client *http.Client, cfg config) summary {
	resultCh := make(chan result, cfg.requests)
	requestCh := make(chan struct{})
	var wg sync.WaitGroup

	for range cfg.concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for range requestCh {
				resultCh <- runRequest(client, cfg)
			}
		}()
	}

	go func() {
		for range cfg.requests {
			requestCh <- struct{}{}
		}
		close(requestCh)
	}()

	summary := newSummary()
	for range cfg.requests {
		res := <-resultCh
		summary.add(res)
	}

	wg.Wait()

	return summary
}
