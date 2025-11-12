// Copyright (c) 2025 Darren Soothill
// Licensed under the MIT License

package storage

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrCircuitBreakerOpen is returned when the circuit breaker is open.
var ErrCircuitBreakerOpen = errors.New("circuit breaker is open")

// State is the state of the circuit breaker.
type State int

const (
	// Closed is the closed state.
	Closed State = iota
	// Open is the open state.
	Open
	// HalfOpen is the half-open state.
	HalfOpen
)

// CircuitBreaker is a circuit breaker implementation.
type CircuitBreaker struct {
	state                    State
	failures                 int
	lastError                error
	lastStateChange          time.Time
	failureThreshold         int
	resetTimeout             time.Duration
	halfOpenSuccesses        int
	halfOpenSuccessThreshold int
	mu                       sync.Mutex
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(failureThreshold int, resetTimeout time.Duration, halfOpenSuccessThreshold int) *CircuitBreaker {
	return &CircuitBreaker{
		state:                    Closed,
		failureThreshold:         failureThreshold,
		resetTimeout:             resetTimeout,
		halfOpenSuccessThreshold: halfOpenSuccessThreshold,
	}
}

// Execute executes the given function with circuit breaker protection.
func (cb *CircuitBreaker) Execute(ctx context.Context, f func(context.Context) error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case Open:
		if time.Since(cb.lastStateChange) > cb.resetTimeout {
			cb.state = HalfOpen
			cb.halfOpenSuccesses = 0
		} else {
			return ErrCircuitBreakerOpen
		}
	case HalfOpen:
		err := f(ctx)
		if err != nil {
			cb.state = Open
			cb.lastStateChange = time.Now()
			cb.lastError = err
			return err
		}
		cb.halfOpenSuccesses++
		if cb.halfOpenSuccesses >= cb.halfOpenSuccessThreshold {
			cb.state = Closed
			cb.failures = 0
		}
		return nil
	}

	err := f(ctx)
	if err != nil {
		cb.failures++
		if cb.failures >= cb.failureThreshold {
			cb.state = Open
			cb.lastStateChange = time.Now()
			cb.lastError = err
		}
		return err
	}

	cb.failures = 0
	return nil
}
