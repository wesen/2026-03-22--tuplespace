package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/manuel/wesen/tuplespace/internal/notify"
	"github.com/manuel/wesen/tuplespace/internal/service"
	"github.com/manuel/wesen/tuplespace/internal/store"
	testpostgres "github.com/manuel/wesen/tuplespace/internal/testutil/postgres"
)

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()

	db := testpostgres.Start(t)
	notifier, err := notify.New(context.Background(), db.URL)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, notifier.Close())
	})

	svc := service.New(db.Pool, store.New(), notifier, service.Options{
		CandidateLimit: 64,
		StartedAt:      time.Now().UTC(),
		ConfigSnapshot: service.RedactedConfigSnapshot(":8080", db.URL, 64, 10*time.Second),
		MigrationFiles: []string{"001_init_tuplespace.sql"},
	})
	return NewHandler(svc)
}

func TestRouterHealthz(t *testing.T) {
	router := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)
	require.Equal(t, http.StatusOK, res.Code)
}

func TestRouterOutAndRd(t *testing.T) {
	router := newTestRouter(t)

	outBody := map[string]any{
		"tuple": map[string]any{
			"fields": []map[string]any{
				{"type": "string", "value": "job"},
				{"type": "int", "value": 42},
			},
		},
	}
	require.Equal(t, http.StatusCreated, performJSON(t, router, "/v1/spaces/jobs/out", outBody).Code)

	rdBody := map[string]any{
		"template": map[string]any{
			"fields": []map[string]any{
				{"kind": "actual", "type": "string", "value": "job"},
				{"kind": "formal", "type": "int", "name": "id"},
			},
		},
		"wait_ms": 0,
	}
	res := performJSON(t, router, "/v1/spaces/jobs/rd", rdBody)
	require.Equal(t, http.StatusOK, res.Code)

	var payload map[string]any
	require.NoError(t, json.NewDecoder(res.Body).Decode(&payload))
	require.Equal(t, true, payload["ok"])
}

func TestRouterBlockingInTimesOut(t *testing.T) {
	router := newTestRouter(t)

	body := map[string]any{
		"template": map[string]any{
			"fields": []map[string]any{
				{"kind": "actual", "type": "string", "value": "job"},
			},
		},
		"wait_ms": 100,
	}
	res := performJSON(t, router, "/v1/spaces/jobs/in", body)
	require.Equal(t, http.StatusRequestTimeout, res.Code)
}

func TestRouterBlockingInSucceedsAfterOut(t *testing.T) {
	router := newTestRouter(t)

	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		body := map[string]any{
			"template": map[string]any{
				"fields": []map[string]any{
					{"kind": "actual", "type": "string", "value": "job"},
				},
			},
			"wait_ms": 2000,
		}
		done <- performJSON(t, router, "/v1/spaces/jobs/in", body)
	}()

	time.Sleep(300 * time.Millisecond)
	outBody := map[string]any{
		"tuple": map[string]any{
			"fields": []map[string]any{
				{"type": "string", "value": "job"},
			},
		},
	}
	require.Equal(t, http.StatusCreated, performJSON(t, router, "/v1/spaces/jobs/out", outBody).Code)

	select {
	case res := <-done:
		require.Equal(t, http.StatusOK, res.Code)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for blocking in response")
	}
}

func TestRouterAdminReadOnlyEndpoints(t *testing.T) {
	router := newTestRouter(t)

	outBody := map[string]any{
		"tuple": map[string]any{
			"fields": []map[string]any{
				{"type": "string", "value": "job"},
			},
		},
	}
	require.Equal(t, http.StatusCreated, performJSON(t, router, "/v1/spaces/jobs/out", outBody).Code)

	spacesRes := performRequest(t, router, http.MethodGet, "/v1/admin/spaces", nil)
	require.Equal(t, http.StatusOK, spacesRes.Code)

	dumpRes := performJSON(t, router, "/v1/admin/dump", map[string]any{
		"filter": map[string]any{"space": "jobs"},
	})
	require.Equal(t, http.StatusOK, dumpRes.Code)

	statsRes := performRequest(t, router, http.MethodGet, "/v1/admin/stats", nil)
	require.Equal(t, http.StatusOK, statsRes.Code)

	configRes := performRequest(t, router, http.MethodGet, "/v1/admin/config", nil)
	require.Equal(t, http.StatusOK, configRes.Code)

	schemaRes := performRequest(t, router, http.MethodGet, "/v1/admin/schema", nil)
	require.Equal(t, http.StatusOK, schemaRes.Code)
}

func performJSON(t *testing.T, handler http.Handler, path string, body any) *httptest.ResponseRecorder {
	return performRequest(t, handler, http.MethodPost, path, body)
}

func performRequest(t *testing.T, handler http.Handler, method string, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, path, nil)
	} else {
		payload, err := json.Marshal(body)
		require.NoError(t, err)
		req = httptest.NewRequest(method, path, bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
	}
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	return res
}
