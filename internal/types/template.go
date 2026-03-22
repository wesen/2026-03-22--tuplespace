package types

type TemplateFieldKind string

const (
	FieldActual TemplateFieldKind = "actual"
	FieldFormal TemplateFieldKind = "formal"
)

type TemplateField struct {
	Kind  TemplateFieldKind `json:"kind"`
	Type  ValueType         `json:"type"`
	Name  string            `json:"name,omitempty"`
	Value any               `json:"value,omitempty"`
}

type Template struct {
	Fields []TemplateField `json:"fields"`
}

type Bindings map[string]any
