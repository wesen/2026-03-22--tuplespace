package httpapi

import "github.com/manuel/wesen/tuplespace/internal/types"

type OutResponse struct {
	OK    bool   `json:"ok"`
	Space string `json:"space"`
	Arity int    `json:"arity"`
}

type ReadResponse struct {
	OK       bool           `json:"ok"`
	Tuple    types.Tuple    `json:"tuple,omitempty"`
	Bindings types.Bindings `json:"bindings,omitempty"`
}

type ErrorEnvelope struct {
	OK    bool         `json:"ok"`
	Error ErrorPayload `json:"error"`
}

type ErrorPayload struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}
