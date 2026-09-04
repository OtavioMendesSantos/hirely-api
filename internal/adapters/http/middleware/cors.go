package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		// Com AllowCredentials=true, não podemos usar "*".
		// Fazemos o reflection do origin se presente (permitindo acesso dinâmico).
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
		} else {
			// Apenas para não quebrar algumas ferramentas antigas,
			// em tese para CLI o CORS nem importa.
			c.Header("Access-Control-Allow-Origin", "http://localhost:4200")
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Header("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization, X-Request-ID, X-Trace-ID")
		c.Header("Access-Control-Expose-Headers", "Authorization, X-Trace-ID")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
