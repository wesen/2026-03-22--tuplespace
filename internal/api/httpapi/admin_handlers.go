package httpapi

import (
	"context"
	"net/http"

	"github.com/manuel/wesen/tuplespace/internal/admin"
	"github.com/manuel/wesen/tuplespace/internal/service"
)

func (h *Handlers) handleAdminSpaces(w http.ResponseWriter, r *http.Request) {
	spaces, err := h.service.Spaces(r.Context())
	if err != nil {
		writeMappedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, SpacesResponse{OK: true, Spaces: spaces})
}

func (h *Handlers) handleAdminDump(w http.ResponseWriter, r *http.Request) {
	h.handleTupleListRequest(w, r, h.service.Dump)
}

func (h *Handlers) handleAdminPeek(w http.ResponseWriter, r *http.Request) {
	h.handleTupleListRequest(w, r, h.service.Peek)
}

func (h *Handlers) handleAdminExport(w http.ResponseWriter, r *http.Request) {
	h.handleTupleListRequest(w, r, h.service.Export)
}

func (h *Handlers) handleAdminStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.Stats(r.Context())
	if err != nil {
		writeMappedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, StatsResponse{OK: true, Stats: stats})
}

func (h *Handlers) handleAdminConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.service.Config(r.Context())
	if err != nil {
		writeMappedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ConfigResponse{OK: true, Config: cfg})
}

func (h *Handlers) handleAdminSchema(w http.ResponseWriter, r *http.Request) {
	schema, err := h.service.Schema(r.Context())
	if err != nil {
		writeMappedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, SchemaResponse{OK: true, Schema: schema})
}

func (h *Handlers) handleAdminTupleGet(w http.ResponseWriter, r *http.Request, tupleID int64) {
	record, found, err := h.service.GetTuple(r.Context(), tupleID)
	if err != nil {
		writeMappedError(w, err)
		return
	}
	if !found {
		writeMappedError(w, service.ErrNotFound)
		return
	}
	writeJSON(w, http.StatusOK, TupleGetResponse{OK: true, Tuple: record})
}

func (h *Handlers) handleAdminTupleDelete(w http.ResponseWriter, r *http.Request, tupleID int64) {
	result, err := h.service.DeleteTuple(r.Context(), tupleID)
	if err != nil {
		writeMappedError(w, err)
		return
	}
	if !result.Deleted {
		writeMappedError(w, service.ErrNotFound)
		return
	}
	writeJSON(w, http.StatusOK, TupleDeleteResponse{OK: true, Result: result})
}

func (h *Handlers) handleAdminPurge(w http.ResponseWriter, r *http.Request) {
	var req FilterRequest
	if err := decodeJSON(r, &req); err != nil {
		writeMappedError(w, err)
		return
	}
	result, err := h.service.Purge(r.Context(), req.Filter)
	if err != nil {
		writeMappedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, PurgeResponse{OK: true, Result: result})
}

func (h *Handlers) handleAdminWaiters(w http.ResponseWriter, r *http.Request) {
	waiters, err := h.service.Waiters(r.Context())
	if err != nil {
		writeMappedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, WaitersResponse{OK: true, Waiters: waiters})
}

func (h *Handlers) handleAdminNotifyTest(w http.ResponseWriter, r *http.Request) {
	var req NotifyTestRequest
	if err := decodeJSON(r, &req); err != nil {
		writeMappedError(w, err)
		return
	}
	result, err := h.service.NotifyTest(r.Context(), req.Space)
	if err != nil {
		writeMappedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, NotifyTestResponse{OK: true, Result: result})
}

func (h *Handlers) handleTupleListRequest(
	w http.ResponseWriter,
	r *http.Request,
	fn func(context.Context, admin.TupleFilter) ([]admin.TupleRecord, error),
) {
	var req FilterRequest
	if err := decodeJSON(r, &req); err != nil {
		writeMappedError(w, err)
		return
	}
	tuples, err := fn(r.Context(), req.Filter)
	if err != nil {
		writeMappedError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, TupleListResponse{OK: true, Tuples: tuples})
}
