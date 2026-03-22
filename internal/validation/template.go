package validation

import (
	"fmt"

	"github.com/manuel/wesen/tuplespace/internal/types"
)

func ValidateTemplate(template types.Template) (types.Template, error) {
	if len(template.Fields) == 0 {
		return types.Template{}, newError("invalid_template", "template must contain at least one field")
	}

	for i, field := range template.Fields {
		if !types.IsSupportedValueType(field.Type) {
			return types.Template{}, newError("unsupported_type", fmt.Sprintf("template field %d uses unsupported type %q", i, field.Type))
		}

		switch field.Kind {
		case types.FieldActual:
			if field.Name != "" {
				return types.Template{}, newError("invalid_template", fmt.Sprintf("actual field %d must not set name", i))
			}
		case types.FieldFormal:
			if field.Name == "" {
				return types.Template{}, newError("invalid_template", fmt.Sprintf("formal field %d must set a non-empty name", i))
			}
			if field.Value != nil {
				return types.Template{}, newError("invalid_template", fmt.Sprintf("formal field %d must not set value", i))
			}
		default:
			return types.Template{}, newError("invalid_template", fmt.Sprintf("template field %d uses invalid kind %q", i, field.Kind))
		}
	}

	normalized, err := types.NormalizeTemplate(template)
	if err != nil {
		return types.Template{}, newError("invalid_template", err.Error())
	}
	return normalized, nil
}
