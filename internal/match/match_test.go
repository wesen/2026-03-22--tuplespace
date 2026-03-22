package match

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/manuel/wesen/tuplespace/internal/types"
)

func TestMatchActualAndFormalFields(t *testing.T) {
	template := types.Template{
		Fields: []types.TemplateField{
			{Kind: types.FieldActual, Type: types.TypeString, Value: "job"},
			{Kind: types.FieldFormal, Type: types.TypeInt, Name: "id"},
		},
	}
	tuple := types.Tuple{
		Fields: []types.TupleField{
			{Type: types.TypeString, Value: "job"},
			{Type: types.TypeInt, Value: int64(42)},
		},
	}

	bindings, ok := Match(template, tuple)
	require.True(t, ok)
	require.Equal(t, types.Bindings{"id": int64(42)}, bindings)
}

func TestMatchRejectsRepeatedFormalNameWithDifferentValues(t *testing.T) {
	template := types.Template{
		Fields: []types.TemplateField{
			{Kind: types.FieldFormal, Type: types.TypeInt, Name: "id"},
			{Kind: types.FieldFormal, Type: types.TypeInt, Name: "id"},
		},
	}
	tuple := types.Tuple{
		Fields: []types.TupleField{
			{Type: types.TypeInt, Value: int64(1)},
			{Type: types.TypeInt, Value: int64(2)},
		},
	}

	_, ok := Match(template, tuple)
	require.False(t, ok)
}

func TestMatchRejectsTypeMismatch(t *testing.T) {
	template := types.Template{
		Fields: []types.TemplateField{
			{Kind: types.FieldActual, Type: types.TypeString, Value: "job"},
		},
	}
	tuple := types.Tuple{
		Fields: []types.TupleField{
			{Type: types.TypeBool, Value: true},
		},
	}

	_, ok := Match(template, tuple)
	require.False(t, ok)
}
