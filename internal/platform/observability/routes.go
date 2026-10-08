package observability

import "strings"

func RoutePattern(path string) string {
	switch {
	case path == "/health":
		return "/health"
	case path == "/ready":
		return "/ready"
	case path == "/metrics":
		return "/metrics"
	case path == "/items":
		return "/items"
	case strings.HasPrefix(path, "/items/"):
		return "/items/{id}"
	default:
		return "unknown"
	}
}
