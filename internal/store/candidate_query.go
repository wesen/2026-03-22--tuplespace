package store

import (
	"fmt"
	"strings"

	"github.com/manuel/wesen/tuplespace/internal/types"
)

func BuildCandidateQuery(space string, template types.Template, limit int, destructive bool) (string, []any, error) {
	if limit <= 0 {
		limit = 64
	}

	var builder strings.Builder
	args := make([]any, 0, len(template.Fields)+2)
	arg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	builder.WriteString("SELECT t.id, t.space, t.fields_json FROM tuples t")
	joinIndex := 0
	for pos, field := range template.Fields {
		if field.Kind != types.FieldActual {
			continue
		}

		alias := fmt.Sprintf("f%d", joinIndex)
		joinIndex++
		builder.WriteString(" JOIN tuple_fields ")
		builder.WriteString(alias)
		builder.WriteString(" ON ")
		builder.WriteString(alias)
		builder.WriteString(".tuple_id = t.id")
		builder.WriteString(" AND ")
		builder.WriteString(alias)
		builder.WriteString(".pos = ")
		builder.WriteString(arg(pos))
		builder.WriteString(" AND ")
		builder.WriteString(alias)
		builder.WriteString(".type = ")
		builder.WriteString(arg(string(field.Type)))

		switch field.Type {
		case types.TypeString:
			builder.WriteString(" AND ")
			builder.WriteString(alias)
			builder.WriteString(".text_val = ")
			builder.WriteString(arg(field.Value))
		case types.TypeInt:
			builder.WriteString(" AND ")
			builder.WriteString(alias)
			builder.WriteString(".int_val = ")
			builder.WriteString(arg(field.Value))
		case types.TypeBool:
			builder.WriteString(" AND ")
			builder.WriteString(alias)
			builder.WriteString(".bool_val = ")
			builder.WriteString(arg(field.Value))
		default:
			return "", nil, fmt.Errorf("unsupported field type %q", field.Type)
		}
	}

	builder.WriteString(" WHERE t.space = ")
	builder.WriteString(arg(space))
	builder.WriteString(" AND t.arity = ")
	builder.WriteString(arg(len(template.Fields)))
	builder.WriteString(" ORDER BY t.id")
	if destructive {
		builder.WriteString(" FOR UPDATE SKIP LOCKED")
	}
	builder.WriteString(" LIMIT ")
	builder.WriteString(arg(limit))

	return builder.String(), args, nil
}
