package main

import (
	"io"
	"net/http"
	"time"
)

type result struct {
	statusCode int
	latency    time.Duration
	err        error
}

func runSequential(client *http.Client, cfg config) []result {
	results := make([]result, 0, cfg.requests)

	for range cfg.requests {
		start := time.Now()
		resp, err := client.Get(cfg.url)
		if err != nil {
			results = append(results, result{
				latency: time.Since(start),
				err:     err,
			})

			continue
		}

		_, readErr := io.Copy(io.Discard, resp.Body)
		closeErr := resp.Body.Close()
		latency := time.Since(start)

		if readErr != nil {
			results = append(results, result{
				latency:    latency,
				statusCode: resp.StatusCode,
				err:        readErr,
			})
			continue
		}

		if closeErr != nil {
			results = append(results, result{
				latency:    latency,
				statusCode: resp.StatusCode,
				err:        closeErr,
			})
			continue
		}

		results = append(results, result{
			latency:    latency,
			statusCode: resp.StatusCode,
		})
	}

	return results
}
