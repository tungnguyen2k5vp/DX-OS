package httpapi

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type principalRateWindow struct {
	startedAt time.Time
	requests  int
}

type principalRateLimiter struct {
	mu          sync.Mutex
	entries     map[string]principalRateWindow
	limit       int
	window      time.Duration
	lastCleanup time.Time
}

func newPrincipalRateLimiter(limit int, window time.Duration) *principalRateLimiter {
	return &principalRateLimiter{entries: make(map[string]principalRateWindow), limit: limit, window: window}
}

func (l *principalRateLimiter) allow(subject string, now time.Time) (bool, int, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.lastCleanup.IsZero() || now.Sub(l.lastCleanup) >= l.window {
		for key, entry := range l.entries {
			if now.Sub(entry.startedAt) >= 2*l.window {
				delete(l.entries, key)
			}
		}
		l.lastCleanup = now
	}

	entry, exists := l.entries[subject]
	if !exists || now.Sub(entry.startedAt) >= l.window {
		l.entries[subject] = principalRateWindow{startedAt: now, requests: 1}
		return true, l.limit - 1, l.window
	}
	retryAfter := l.window - now.Sub(entry.startedAt)
	if entry.requests >= l.limit {
		return false, 0, retryAfter
	}
	entry.requests++
	l.entries[subject] = entry
	return true, l.limit - entry.requests, retryAfter
}

func (a *api) principalRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := principalFromContext(r.Context())
		if !ok || principal.Subject == "" {
			writeProblem(w, r, http.StatusUnauthorized, "unauthenticated", "Authentication required", "A valid access token is required.")
			return
		}
		allowed, remaining, resetAfter := a.rateLimiter.allow(principal.Subject, time.Now())
		seconds := int(math.Ceil(resetAfter.Seconds()))
		if seconds < 1 {
			seconds = 1
		}
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(a.rateLimiter.limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.Itoa(seconds))
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(seconds))
			writeProblem(w, r, http.StatusTooManyRequests, "rate-limit-exceeded", "Too many requests", "The authenticated user has exceeded the API request limit. Retry later.")
			return
		}
		next.ServeHTTP(w, r)
	})
}
