package httpapi

import (
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func withAccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &responseRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
		}
		started := time.Now()

		next.ServeHTTP(recorder, r)

		event := log.Info()
		switch {
		case recorder.status >= 500:
			event = log.Error()
		case recorder.status >= 400:
			event = log.Warn()
		}

		event.
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", recorder.status).
			Dur("duration", time.Since(started)).
			Str("remote_addr", r.RemoteAddr).
			Msg("http request")
	})
}
