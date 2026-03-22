package cmds

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/manuel/wesen/tuplespace/internal/types"
)

func LoadTuple(path string) (types.Tuple, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return types.Tuple{}, fmt.Errorf("read tuple file: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()

	var tuple types.Tuple
	if err := decoder.Decode(&tuple); err != nil {
		return types.Tuple{}, fmt.Errorf("decode tuple file: %w", err)
	}
	normalized, err := types.NormalizeTuple(tuple)
	if err != nil {
		return types.Tuple{}, fmt.Errorf("normalize tuple file: %w", err)
	}
	return normalized, nil
}

func LoadTemplate(path string) (types.Template, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return types.Template{}, fmt.Errorf("read template file: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()

	var template types.Template
	if err := decoder.Decode(&template); err != nil {
		return types.Template{}, fmt.Errorf("decode template file: %w", err)
	}
	normalized, err := types.NormalizeTemplate(template)
	if err != nil {
		return types.Template{}, fmt.Errorf("normalize template file: %w", err)
	}
	return normalized, nil
}

func LoadTupleInput(path string, spec string) (types.Tuple, error) {
	switch {
	case path != "" && spec != "":
		return types.Tuple{}, fmt.Errorf("provide either tuple-file or tuple-spec, not both")
	case path != "":
		return LoadTuple(path)
	case spec != "":
		return ParseTupleSpec(spec)
	default:
		return types.Tuple{}, fmt.Errorf("one of tuple-file or tuple-spec is required")
	}
}

func LoadTemplateInput(path string, spec string) (types.Template, error) {
	switch {
	case path != "" && spec != "":
		return types.Template{}, fmt.Errorf("provide either template-json-file or template-spec, not both")
	case path != "":
		return LoadTemplate(path)
	case spec != "":
		return ParseTemplateSpec(spec)
	default:
		return types.Template{}, fmt.Errorf("one of template-json-file or template-spec is required")
	}
}

func ParseTupleSpec(spec string) (types.Tuple, error) {
	parts, err := splitSpecFields(spec)
	if err != nil {
		return types.Tuple{}, err
	}

	tuple := types.Tuple{Fields: make([]types.TupleField, len(parts))}
	for i, part := range parts {
		valueType, value, err := parseLiteralToken(part)
		if err != nil {
			return types.Tuple{}, fmt.Errorf("parse tuple field %d: %w", i, err)
		}
		tuple.Fields[i] = types.TupleField{
			Type:  valueType,
			Value: value,
		}
	}

	normalized, err := types.NormalizeTuple(tuple)
	if err != nil {
		return types.Tuple{}, fmt.Errorf("normalize tuple spec: %w", err)
	}
	return normalized, nil
}

func ParseTemplateSpec(spec string) (types.Template, error) {
	parts, err := splitSpecFields(spec)
	if err != nil {
		return types.Template{}, err
	}

	template := types.Template{Fields: make([]types.TemplateField, len(parts))}
	for i, part := range parts {
		field, err := parseTemplateFieldToken(part)
		if err != nil {
			return types.Template{}, fmt.Errorf("parse template field %d: %w", i, err)
		}
		template.Fields[i] = field
	}

	normalized, err := types.NormalizeTemplate(template)
	if err != nil {
		return types.Template{}, fmt.Errorf("normalize template spec: %w", err)
	}
	return normalized, nil
}

func splitSpecFields(spec string) ([]string, error) {
	trimmed := strings.TrimSpace(spec)
	if trimmed == "" {
		return nil, fmt.Errorf("spec must not be empty")
	}

	if strings.HasPrefix(trimmed, "(") && strings.HasSuffix(trimmed, ")") {
		trimmed = strings.TrimSpace(trimmed[1 : len(trimmed)-1])
	}
	if trimmed == "" {
		return nil, fmt.Errorf("spec must contain at least one field")
	}

	parts := []string{}
	var current strings.Builder
	inQuotes := false
	escaped := false

	for _, r := range trimmed {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\' && inQuotes:
			current.WriteRune(r)
			escaped = true
		case r == '"':
			current.WriteRune(r)
			inQuotes = !inQuotes
		case r == ',' && !inQuotes:
			part := strings.TrimSpace(current.String())
			if part == "" {
				return nil, fmt.Errorf("spec contains an empty field")
			}
			parts = append(parts, part)
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}

	if inQuotes {
		return nil, fmt.Errorf("spec contains an unterminated quoted string")
	}

	part := strings.TrimSpace(current.String())
	if part == "" {
		return nil, fmt.Errorf("spec contains an empty field")
	}
	parts = append(parts, part)
	return parts, nil
}

func parseTemplateFieldToken(token string) (types.TemplateField, error) {
	if strings.HasPrefix(token, "?") {
		return parseFormalToken(token)
	}

	valueType, value, err := parseLiteralToken(token)
	if err != nil {
		return types.TemplateField{}, err
	}
	return types.TemplateField{
		Kind:  types.FieldActual,
		Type:  valueType,
		Value: value,
	}, nil
}

func parseFormalToken(token string) (types.TemplateField, error) {
	rest := strings.TrimSpace(strings.TrimPrefix(token, "?"))
	name, typeName, ok := strings.Cut(rest, ":")
	if !ok {
		return types.TemplateField{}, fmt.Errorf("formal field %q must use ?name:type", token)
	}

	name = strings.TrimSpace(name)
	typeName = strings.TrimSpace(typeName)
	if name == "" {
		return types.TemplateField{}, fmt.Errorf("formal field %q must set a name", token)
	}
	if typeName == "" {
		return types.TemplateField{}, fmt.Errorf("formal field %q must set a type", token)
	}

	valueType := types.ValueType(typeName)
	if !types.IsSupportedValueType(valueType) {
		return types.TemplateField{}, fmt.Errorf("formal field %q uses unsupported type %q", token, typeName)
	}

	return types.TemplateField{
		Kind: types.FieldFormal,
		Type: valueType,
		Name: name,
	}, nil
}

func parseLiteralToken(token string) (types.ValueType, any, error) {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return "", nil, fmt.Errorf("literal must not be empty")
	}

	if strings.HasPrefix(trimmed, "\"") {
		if !strings.HasSuffix(trimmed, "\"") || len(trimmed) < 2 {
			return "", nil, fmt.Errorf("string literal %q must be double-quoted", token)
		}
		value, err := strconv.Unquote(trimmed)
		if err != nil {
			return "", nil, fmt.Errorf("unquote string literal %q: %w", token, err)
		}
		return types.TypeString, value, nil
	}

	switch trimmed {
	case "true":
		return types.TypeBool, true, nil
	case "false":
		return types.TypeBool, false, nil
	}

	if i, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return types.TypeInt, i, nil
	}

	return types.TypeString, trimmed, nil
}
