package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// FixedWindowLimiter é um limitador de taxa simples por chave (ex.: IP) com
// janela fixa. É uma proteção barata contra abuso em endpoints sensíveis como
// o MCP; para casos de uso maiores, troque por um solução distribuída.
type FixedWindowLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	windows map[string]*windowEntry
	stop    chan struct{}
	done    chan struct{}
}

type windowEntry struct {
	count   int
	resetAt time.Time
}

// NewFixedWindowLimiter cria um limitador que permite até `limit` requisições
// por `window` por chave. Um janitor interno remove entradas expiradas para
// evitar crescimento de memória com muitos IPs distintos.
func NewFixedWindowLimiter(limit int, window time.Duration) *FixedWindowLimiter {
	l := &FixedWindowLimiter{
		limit:   limit,
		window:  window,
		windows: make(map[string]*windowEntry),
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
	go l.janitor()
	return l
}

// Stop encerra o janitor interno. Chame apenas em shutdown.
func (l *FixedWindowLimiter) Stop() {
	select {
	case <-l.stop:
		return // já parado
	default:
	}
	close(l.stop)
	<-l.done
}

// Allow reporta se a chave ainda está dentro do limite da janela vigente.
func (l *FixedWindowLimiter) Allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.windows[key]
	if !ok || now.After(entry.resetAt) {
		l.windows[key] = &windowEntry{count: 1, resetAt: now.Add(l.window)}
		return true
	}
	entry.count++
	return entry.count <= l.limit
}

func (l *FixedWindowLimiter) janitor() {
	defer close(l.done)
	ticker := time.NewTicker(max(l.window/2, time.Second))
	defer ticker.Stop()
	for {
		select {
		case <-l.stop:
			return
		case now := <-ticker.C:
			l.mu.Lock()
			for k, entry := range l.windows {
				if now.After(entry.resetAt) {
					delete(l.windows, k)
				}
			}
			l.mu.Unlock()
		}
	}
}

// RateLimit aplica o limitador por ClientIP.
func RateLimit(limiter *FixedWindowLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !limiter.Allow(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many requests",
				"code":  "RATE_LIMITED",
			})
			return
		}
		c.Next()
	}
}
