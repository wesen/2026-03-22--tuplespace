package types

import (
	"encoding/json"
	"fmt"
	"math"
)

func IsSupportedValueType(valueType ValueType) bool {
	switch valueType {
	case TypeString, TypeInt, TypeBool:
		return true
	default:
		return false
	}
}

func NormalizeValue(valueType ValueType, raw any) (any, error) {
	switch valueType {
	case TypeString:
		s, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", raw)
		}
		return s, nil
	case TypeInt:
		return normalizeInt(raw)
	case TypeBool:
		b, ok := raw.(bool)
		if !ok {
			return nil, fmt.Errorf("expected bool, got %T", raw)
		}
		return b, nil
	default:
		return nil, fmt.Errorf("unsupported value type %q", valueType)
	}
}

func normalizeInt(raw any) (int64, error) {
	switch v := raw.(type) {
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		if uint64(v) > math.MaxInt64 {
			return 0, fmt.Errorf("uint value %d overflows int64", v)
		}
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		if v > math.MaxInt64 {
			return 0, fmt.Errorf("uint64 value %d overflows int64", v)
		}
		return int64(v), nil
	case float32:
		if math.Trunc(float64(v)) != float64(v) {
			return 0, fmt.Errorf("expected integral float32, got %v", v)
		}
		return int64(v), nil
	case float64:
		if math.Trunc(v) != v {
			return 0, fmt.Errorf("expected integral float64, got %v", v)
		}
		return int64(v), nil
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return 0, fmt.Errorf("parse int json.Number: %w", err)
		}
		return i, nil
	default:
		return 0, fmt.Errorf("expected int, got %T", raw)
	}
}

func NormalizeTuple(tuple Tuple) (Tuple, error) {
	normalized := Tuple{Fields: make([]TupleField, len(tuple.Fields))}
	for i, field := range tuple.Fields {
		value, err := NormalizeValue(field.Type, field.Value)
		if err != nil {
			return Tuple{}, fmt.Errorf("field %d: %w", i, err)
		}
		normalized.Fields[i] = TupleField{
			Type:  field.Type,
			Value: value,
		}
	}
	return normalized, nil
}

func NormalizeTemplate(template Template) (Template, error) {
	normalized := Template{Fields: make([]TemplateField, len(template.Fields))}
	for i, field := range template.Fields {
		next := TemplateField{
			Kind: field.Kind,
			Type: field.Type,
			Name: field.Name,
		}
		if field.Kind == FieldActual {
			value, err := NormalizeValue(field.Type, field.Value)
			if err != nil {
				return Template{}, fmt.Errorf("field %d: %w", i, err)
			}
			next.Value = value
		}
		normalized.Fields[i] = next
	}
	return normalized, nil
}
