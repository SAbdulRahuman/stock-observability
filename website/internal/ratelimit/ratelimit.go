// Package ratelimit limits requests per client IP with token buckets.
package ratelimit

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	maxClients = 10_000
	idleAfter  = 10 * time.Minute
)

// Limiter keeps one token bucket per key.
type Limiter struct {
	limit   rate.Limit
	burst   int
	now     func() time.Time
	mu      sync.Mutex
	clients map[string]*client
}

type client struct {
	lim  *rate.Limiter
	seen time.Time
}

// New allows burst requests at once and r per second after that, per key.
func New(r rate.Limit, burst int) *Limiter {
	return &Limiter{limit: r, burst: burst, now: time.Now, clients: map[string]*client{}}
}

// Allow reports whether one more request from key is allowed now.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if len(l.clients) >= maxClients {
		l.prune(now)
	}
	c, ok := l.clients[key]
	if !ok {
		c = &client{lim: rate.NewLimiter(l.limit, l.burst)}
		l.clients[key] = c
	}
	c.seen = now
	return c.lim.AllowN(now, 1)
}

func (l *Limiter) prune(now time.Time) {
	for k, c := range l.clients {
		if now.Sub(c.seen) > idleAfter {
			delete(l.clients, k)
		}
	}
	// Under a flood of distinct IPs, forgetting everyone is better than unbounded memory.
	if len(l.clients) >= maxClients {
		clear(l.clients)
	}
}

// ClientIP is the TCP peer address; the server is exposed directly, so forwarding headers are not trusted.
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
