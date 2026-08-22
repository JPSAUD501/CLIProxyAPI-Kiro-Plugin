package kiro

import (
	"math/rand"
	"sync"
	"time"
)

// Jitter configuration constants
const (
	// JitterPercent is the default percentage of jitter to apply (±30%)
	JitterPercent = 0.30
)

var (
	jitterRand     *rand.Rand
	jitterRandOnce sync.Once
	jitterMu       sync.Mutex
)

// initJitterRand initializes the random number generator for jitter calculations.
// Uses a time-based seed for unpredictable but reproducible randomness.
func initJitterRand() {
	jitterRandOnce.Do(func() {
		jitterRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	})
}

// RandomDelay generates a random delay between min and max duration.
// Thread-safe implementation using mutex protection.
func RandomDelay(min, max time.Duration) time.Duration {
	initJitterRand()
	jitterMu.Lock()
	defer jitterMu.Unlock()

	if min >= max {
		return min
	}

	rangeMs := max.Milliseconds() - min.Milliseconds()
	randomMs := jitterRand.Int63n(rangeMs)
	return min + time.Duration(randomMs)*time.Millisecond
}

// JitterDelay adds jitter to a base delay.
// Applies ±jitterPercent variation to the base delay.
// For example, JitterDelay(1*time.Second, 0.30) returns a value between 700ms and 1300ms.
func JitterDelay(baseDelay time.Duration, jitterPercent float64) time.Duration {
	initJitterRand()
	jitterMu.Lock()
	defer jitterMu.Unlock()

	if jitterPercent <= 0 || jitterPercent > 1 {
		jitterPercent = JitterPercent
	}

	// Calculate jitter range: base * jitterPercent
	jitterRange := float64(baseDelay) * jitterPercent

	// Generate random value in range [-jitterRange, +jitterRange]
	jitter := (jitterRand.Float64()*2 - 1) * jitterRange

	result := time.Duration(float64(baseDelay) + jitter)
	if result < 0 {
		return 0
	}
	return result
}

// JitterDelayDefault applies the default ±30% jitter to a base delay.
func JitterDelayDefault(baseDelay time.Duration) time.Duration {
	return JitterDelay(baseDelay, JitterPercent)
}

// ExponentialBackoffWithJitter calculates retry delay using exponential backoff with jitter.
// Formula: min(baseDelay * 2^attempt + jitter, maxDelay)
// This helps prevent thundering herd problem when multiple clients retry simultaneously.
func ExponentialBackoffWithJitter(attempt int, baseDelay, maxDelay time.Duration) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	// Calculate exponential backoff: baseDelay * 2^attempt
	backoff := baseDelay * time.Duration(1<<uint(attempt))
	if backoff > maxDelay {
		backoff = maxDelay
	}

	// Add ±30% jitter
	return JitterDelay(backoff, JitterPercent)
}
