package http

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
)

// sourceContextKey is an unexported type used to store the client source in a context.
type sourceContextKey struct{}

// clientSourceMiddleware reads the X-Client request header (or falls back to User-Agent)
// and injects the detected source string into the request context so Huma handlers can
// retrieve it without coupling to Gin.
//
// Known sources: "web", "android", "api" (default for direct API callers).
func clientSourceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		src := detectSource(c.GetHeader("X-Client"), c.GetHeader("User-Agent"))
		c.Request = c.Request.WithContext(
			context.WithValue(c.Request.Context(), sourceContextKey{}, src),
		)
		c.Next()
	}
}

// detectSource determines the client source from explicit and implicit signals.
func detectSource(xClient, userAgent string) string {
	switch strings.ToLower(strings.TrimSpace(xClient)) {
	case "android":
		return "android"
	case "web":
		return "web"
	case "api":
		return "api"
	}
	// Fallback: inspect User-Agent for Android app signals.
	ua := strings.ToLower(userAgent)
	if strings.Contains(ua, "android") || strings.Contains(ua, "dalvik") {
		return "android"
	}
	return "api"
}

// sourceFromContext returns the client source injected by clientSourceMiddleware,
// defaulting to "api" when the value is absent.
func sourceFromContext(ctx context.Context) string {
	if s, ok := ctx.Value(sourceContextKey{}).(string); ok && s != "" {
		return s
	}
	return "api"
}
