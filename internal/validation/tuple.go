package validation

import (
	"fmt"

	"github.com/manuel/wesen/tuplespace/internal/types"
)

func ValidateTuple(tuple types.Tuple) (types.Tuple, error) {
	if len(tuple.Fields) == 0 {
		return types.Tuple{}, newError("invalid_tuple", "tuple must contain at least one field")
	}

	for i, field := range tuple.Fields {
		if !types.IsSupportedValueType(field.Type) {
			return types.Tuple{}, newError("unsupported_type", fmt.Sprintf("tuple field %d uses unsupported type %q", i, field.Type))
		}
	}

	normalized, err := types.NormalizeTuple(tuple)
	if err != nil {
		return types.Tuple{}, newError("invalid_tuple", err.Error())
	}
	return normalized, nil
}
