package httpapi

import "github.com/manuel/wesen/tuplespace/internal/types"

type OutRequest struct {
	Tuple types.Tuple `json:"tuple"`
}

type ReadRequest struct {
	Template types.Template `json:"template"`
	WaitMS   int64          `json:"wait_ms"`
}
