package auth

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// clientHistory almacena las marcas de tiempo de las peticiones de una IP.
type clientHistory struct {
	timestamps []time.Time
}

// IPRateLimiter implementa un limitador de tasa deslizante por IP en memoria.
type IPRateLimiter struct {
	mu      sync.Mutex
	clients map[string]*clientHistory
	limit   int
	window  time.Duration
}

// NewIPRateLimiter crea una nueva instancia de limitador por IP.
func NewIPRateLimiter(limit int, window time.Duration) *IPRateLimiter {
	limiter := &IPRateLimiter{
		clients: make(map[string]*clientHistory),
		limit:   limit,
		window:  window,
	}

	// Goroutine en segundo plano para limpiar IPs inactivas y evitar fugas de memoria
	go func() {
		ticker := time.NewTicker(window * 2)
		for range ticker.C {
			limiter.cleanup()
		}
	}()

	return limiter
}

func (limiter *IPRateLimiter) cleanup() {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	threshold := time.Now().Add(-limiter.window)
	for ip, history := range limiter.clients {
		valid := history.timestamps[:0]
		for _, t := range history.timestamps {
			if t.After(threshold) {
				valid = append(valid, t)
			}
		}
		if len(valid) == 0 {
			delete(limiter.clients, ip)
		} else {
			history.timestamps = valid
		}
	}
}

// Allow verifica si la IP tiene permiso para realizar una petición bajo la ventana deslizante.
func (limiter *IPRateLimiter) Allow(ip string) bool {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	now := time.Now()
	threshold := now.Add(-limiter.window)

	history, exists := limiter.clients[ip]
	if !exists {
		history = &clientHistory{timestamps: make([]time.Time, 0, limiter.limit)}
		limiter.clients[ip] = history
	}

	// Filtrar peticiones fuera de la ventana actual
	valid := history.timestamps[:0]
	for _, t := range history.timestamps {
		if t.After(threshold) {
			valid = append(valid, t)
		}
	}
	history.timestamps = valid

	if len(history.timestamps) >= limiter.limit {
		return false
	}

	history.timestamps = append(history.timestamps, now)
	return true
}

// Middleware retorna un gin.HandlerFunc que intercepta peticiones que excedan la cuota.
func (limiter *IPRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.Allow(ip) {
			c.Header("Retry-After", "60")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests, please try again later",
			})
			return
		}
		c.Next()
	}
}

// RateLimitMiddleware crea un middleware de Gin configurado a 5 peticiones por minuto por IP.
func RateLimitMiddleware() gin.HandlerFunc {
	limiter := NewIPRateLimiter(5, time.Minute)
	return limiter.Middleware()
}
