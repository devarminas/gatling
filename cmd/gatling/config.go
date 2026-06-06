package main

import "fmt"

type config struct {
	requests    int
	concurrency int
	url         string
}

func (c config) validate() error {
	if c.requests <= 0 {
		return fmt.Errorf("-n must be greater than 0, got %d", c.requests)
	}
	if c.concurrency <= 0 {
		return fmt.Errorf("-c must be greater than 0, got %d", c.concurrency)
	}
	if c.concurrency > c.requests {
		return fmt.Errorf("-c (%d) cannot exceed -n (%d)", c.concurrency, c.requests)
	}
	return nil
}
