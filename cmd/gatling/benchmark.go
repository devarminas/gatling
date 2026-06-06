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

func runConcurrent(client *http.Client, cfg config) []result {
	jobs := make(chan int)
	results := make([]result, cfg.requests)
	var wg sync.WaitGroup

	for range cfg.concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for job := range jobs {
				results[job] = runRequest(client, cfg)
			}
		}()
	}

	for job := range cfg.requests {
		jobs <- job
	}
	close(jobs)
	wg.Wait()

	return results
}
