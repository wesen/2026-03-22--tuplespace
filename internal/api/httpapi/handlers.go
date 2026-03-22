package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/manuel/wesen/tuplespace/internal/service"
	"github.com/manuel/wesen/tuplespace/internal/types"
)

type Handlers struct {
	service service.TupleSpace
}

func NewHandlers(service service.TupleSpace) *Handlers {
	return &Handlers{service: service}
}

func (h *Handlers) handleOut(w http.ResponseWriter, r *http.Request, space string) {
	var req OutRequest
	if err := decodeJSON(r, &req); err != nil {
		writeMappedError(w, err)
		return
	}

	if err := h.service.Out(r.Context(), space, req.Tuple); err != nil {
		writeMappedError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, OutResponse{
		OK:    true,
		Space: space,
		Arity: len(req.Tuple.Fields),
	})
}

func (h *Handlers) handleRd(w http.ResponseWriter, r *http.Request, space string) {
	h.handleRead(w, r, space, false)
}

func (h *Handlers) handleIn(w http.ResponseWriter, r *http.Request, space string) {
	h.handleRead(w, r, space, true)
}

func (h *Handlers) handleRead(w http.ResponseWriter, r *http.Request, space string, destructive bool) {
	var req ReadRequest
	if err := decodeJSON(r, &req); err != nil {
		writeMappedError(w, err)
		return
	}
	wait := time.Duration(req.WaitMS) * time.Millisecond

	var (
		tuple    types.Tuple
		bindings types.Bindings
		err      error
	)
	if destructive {
		tuple, bindings, err = h.service.In(r.Context(), space, req.Template, wait)
	} else {
		tuple, bindings, err = h.service.Rd(r.Context(), space, req.Template, wait)
	}
	if err != nil {
		writeMappedError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, ReadResponse{
		OK:       true,
		Tuple:    tuple,
		Bindings: bindings,
	})
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()

	body := new(bytes.Buffer)
	if _, err := body.ReadFrom(r.Body); err != nil {
		return fmt.Errorf("read request body: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(body.Bytes()))
	decoder.UseNumber()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("decode request body: %w", err)
	}
	return nil
}

func writeMappedError(w http.ResponseWriter, err error) {
	status, payload := mapError(err)
	writeJSON(w, status, ErrorEnvelope{
		OK:    false,
		Error: payload,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
