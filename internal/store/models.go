package store

import "github.com/manuel/wesen/tuplespace/internal/types"

type StoredTuple struct {
	ID    int64
	Space string
	Tuple types.Tuple
}
