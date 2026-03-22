package match

import "github.com/manuel/wesen/tuplespace/internal/types"

func Match(template types.Template, tuple types.Tuple) (types.Bindings, bool) {
	if len(template.Fields) != len(tuple.Fields) {
		return nil, false
	}

	bindings := types.Bindings{}
	for i := range template.Fields {
		templateField := template.Fields[i]
		tupleField := tuple.Fields[i]

		if templateField.Type != tupleField.Type {
			return nil, false
		}

		switch templateField.Kind {
		case types.FieldActual:
			if !EqualValue(templateField.Type, templateField.Value, tupleField.Value) {
				return nil, false
			}
		case types.FieldFormal:
			if existing, ok := bindings[templateField.Name]; ok {
				if !EqualValue(templateField.Type, existing, tupleField.Value) {
					return nil, false
				}
				continue
			}
			bindings[templateField.Name] = tupleField.Value
		default:
			return nil, false
		}
	}

	return bindings, true
}
