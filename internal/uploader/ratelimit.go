package uploader

import (
	"context"
	"time"
)

type RateLimiter struct {
	tokens         chan struct{}
	tokensPerSec  int
	maxBurst      int
	tokenInterval time.Duration
}

func NewRateLimiter(tokensPerSec, maxBurst int) *RateLimiter {
	r := &RateLimiter{
		tokens:         make(chan struct{}, maxBurst),
		tokensPerSec:  tokensPerSec,
		maxBurst:      maxBurst,
		tokenInterval: time.Second / time.Duration(tokensPerSec),
	}

	// Start token generator
	go r.generateTokens()
	return r
}

func (r *RateLimiter) generateTokens() {
	ticker := time.NewTicker(r.tokenInterval)
	defer ticker.Stop()

	for range ticker.C {
		select {
		case r.tokens <- struct{}{}:
			// Token added
		default:
			// Buffer full, skip
		}
	}
}

func (r *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-r.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
