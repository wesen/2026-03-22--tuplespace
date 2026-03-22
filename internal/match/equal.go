package match

import "github.com/manuel/wesen/tuplespace/internal/types"

func EqualValue(valueType types.ValueType, left, right any) bool {
	normalizedLeft, err := types.NormalizeValue(valueType, left)
	if err != nil {
		return false
	}
	normalizedRight, err := types.NormalizeValue(valueType, right)
	if err != nil {
		return false
	}
	return normalizedLeft == normalizedRight
}
