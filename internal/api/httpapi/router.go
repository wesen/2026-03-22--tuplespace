package httpapi

import (
	"net/http"
	"strconv"
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
	mux.HandleFunc("/v1/admin/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/admin/spaces":
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			handlers.handleAdminSpaces(w, r)
		case r.URL.Path == "/v1/admin/dump":
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			handlers.handleAdminDump(w, r)
		case r.URL.Path == "/v1/admin/stats":
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			handlers.handleAdminStats(w, r)
		case r.URL.Path == "/v1/admin/config":
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			handlers.handleAdminConfig(w, r)
		case r.URL.Path == "/v1/admin/schema":
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			handlers.handleAdminSchema(w, r)
		case r.URL.Path == "/v1/admin/peek":
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			handlers.handleAdminPeek(w, r)
		case r.URL.Path == "/v1/admin/export":
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			handlers.handleAdminExport(w, r)
		case r.URL.Path == "/v1/admin/purge":
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			handlers.handleAdminPurge(w, r)
		case r.URL.Path == "/v1/admin/waiters":
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			handlers.handleAdminWaiters(w, r)
		case r.URL.Path == "/v1/admin/notify-test":
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			handlers.handleAdminNotifyTest(w, r)
		default:
			tupleID, ok := parseAdminTuplePath(r.URL.Path)
			if !ok {
				http.NotFound(w, r)
				return
			}
			switch r.Method {
			case http.MethodGet:
				handlers.handleAdminTupleGet(w, r, tupleID)
			case http.MethodDelete:
				handlers.handleAdminTupleDelete(w, r, tupleID)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		}
	})
	return withAccessLog(mux)
}

func parseSpacePath(path string) (space string, operation string, ok bool) {
	trimmed := strings.TrimPrefix(path, "/v1/spaces/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func parseAdminTuplePath(path string) (int64, bool) {
	trimmed := strings.TrimPrefix(path, "/v1/admin/tuples/")
	if trimmed == "" || strings.Contains(trimmed, "/") {
		return 0, false
	}
	tupleID, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return 0, false
	}
	return tupleID, true
}
