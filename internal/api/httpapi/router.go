package httpapi

import (
	"net/http"
	"strings"

	"github.com/manuel/wesen/tuplespace/internal/service"
)

func NewHandler(service service.TupleSpace) http.Handler {
	handlers := NewHandlers(service)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})
	mux.HandleFunc("/v1/spaces/", func(w http.ResponseWriter, r *http.Request) {
		space, operation, ok := parseSpacePath(r.URL.Path)
		if !ok {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		switch operation {
		case "out":
			handlers.handleOut(w, r, space)
		case "rd":
			handlers.handleRd(w, r, space)
		case "in":
			handlers.handleIn(w, r, space)
		default:
			http.NotFound(w, r)
		}
	})
	return mux
}

func parseSpacePath(path string) (space string, operation string, ok bool) {
	trimmed := strings.TrimPrefix(path, "/v1/spaces/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}
