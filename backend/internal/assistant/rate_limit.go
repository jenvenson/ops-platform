// Copyright (c) 2026 OPS Platform Contributors.
// SPDX-License-Identifier: MIT

package assistant

import (
	"sync"
	"time"
)

type rateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	buckets map[string][]time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	if limit <= 0 {
		limit = 30
	}
	if window <= 0 {
		window = time.Minute
	}
	return &rateLimiter{
		limit:   limit,
		window:  window,
		buckets: make(map[string][]time.Time),
	}
}

func (r *rateLimiter) Allow(key string) bool {
	if r == nil || key == "" {
		return true
	}

	now := time.Now()
	cutoff := now.Add(-r.window)

	r.mu.Lock()
	defer r.mu.Unlock()

	timestamps := r.buckets[key]
	filtered := timestamps[:0]
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			filtered = append(filtered, ts)
		}
	}

	if len(filtered) >= r.limit {
		r.buckets[key] = filtered
		return false
	}

	filtered = append(filtered, now)
	r.buckets[key] = filtered
	return true
}