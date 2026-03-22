package types

type ValueType string

const (
	TypeString ValueType = "string"
	TypeInt    ValueType = "int"
	TypeBool   ValueType = "bool"
)

type TupleField struct {
	Type  ValueType `json:"type"`
	Value any       `json:"value"`
}

type Tuple struct {
	Fields []TupleField `json:"fields"`
}
