package gateway

import (
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	defaultAuthFailureLimit  = 5
	defaultAuthFailureWindow = time.Minute
	defaultAuthBlockDuration = time.Minute
)

type mcpAuthLimiter struct {
	mu            sync.Mutex
	now           func() time.Time
	maxFailures   int
	failureWindow time.Duration
	blockDuration time.Duration
	failures      map[string]authFailureState
}

type authFailureState struct {
	firstFailure time.Time
	count        int
	blockedUntil time.Time
}

func newMCPAuthLimiter() *mcpAuthLimiter {
	return &mcpAuthLimiter{
		now:           time.Now,
		maxFailures:   defaultAuthFailureLimit,
		failureWindow: defaultAuthFailureWindow,
		blockDuration: defaultAuthBlockDuration,
		failures:      make(map[string]authFailureState),
	}
}

func (l *mcpAuthLimiter) RecordSuccess(remoteAddr string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.failures, authRemoteKey(remoteAddr))
}

func (l *mcpAuthLimiter) RecordFailure(remoteAddr string) bool {
	key := authRemoteKey(remoteAddr)
	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	l.compactLocked(now)

	state := l.failures[key]
	if !state.blockedUntil.IsZero() && now.Before(state.blockedUntil) {
		return true
	}
	if state.firstFailure.IsZero() || now.Sub(state.firstFailure) > l.failureWindow {
		state = authFailureState{firstFailure: now}
	}
	state.count++
	if state.count >= l.maxFailures {
		state.blockedUntil = now.Add(l.blockDuration)
	}
	l.failures[key] = state
	return !state.blockedUntil.IsZero() && now.Before(state.blockedUntil)
}

func (l *mcpAuthLimiter) compactLocked(now time.Time) {
	for key, state := range l.failures {
		if !state.blockedUntil.IsZero() {
			if now.After(state.blockedUntil) {
				delete(l.failures, key)
			}
			continue
		}
		if !state.firstFailure.IsZero() && now.Sub(state.firstFailure) > l.failureWindow {
			delete(l.failures, key)
		}
	}
}

func authRemoteKey(remoteAddr string) string {
	trimmed := strings.TrimSpace(remoteAddr)
	if trimmed == "" {
		return "unknown"
	}
	host, _, err := net.SplitHostPort(trimmed)
	if err == nil && host != "" {
		return host
	}
	return trimmed
}

func defaultLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stderr, nil))
}
