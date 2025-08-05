package httpapi

import (
	"net/http"
)

// FileServerWithIndexFallback creates a file server that serves the given filesystem
// and falls back to index.html for any path that doesn't match a file
func FileServerWithIndexFallback(chatBasePath string) http.Handler {
	// UI is removed in this custom build
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w,
			"UI is disabled in this custom AgentAPI build. This is API-only version for Claude Code Studio.",
			http.StatusNotFound)
	})
}
