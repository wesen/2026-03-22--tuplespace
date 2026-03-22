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

	svc := service.New(db.Pool, store.New(), notifier, 64)
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

func performJSON(t *testing.T, handler http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	return res
}
