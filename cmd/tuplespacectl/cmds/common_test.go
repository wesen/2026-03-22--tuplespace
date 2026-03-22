package cmds

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/manuel/wesen/tuplespace/internal/types"
)

func TestParseTupleSpec(t *testing.T) {
	tuple, err := ParseTupleSpec(`("job with spaces",42,true,worker-1,"123")`)
	require.NoError(t, err)
	require.Equal(t, types.Tuple{
		Fields: []types.TupleField{
			{Type: types.TypeString, Value: "job with spaces"},
			{Type: types.TypeInt, Value: int64(42)},
			{Type: types.TypeBool, Value: true},
			{Type: types.TypeString, Value: "worker-1"},
			{Type: types.TypeString, Value: "123"},
		},
	}, tuple)
}

func TestParseTemplateSpec(t *testing.T) {
	template, err := ParseTemplateSpec(`job,?id:int,?ready:bool,"123"`)
	require.NoError(t, err)
	require.Equal(t, types.Template{
		Fields: []types.TemplateField{
			{Kind: types.FieldActual, Type: types.TypeString, Value: "job"},
			{Kind: types.FieldFormal, Type: types.TypeInt, Name: "id"},
			{Kind: types.FieldFormal, Type: types.TypeBool, Name: "ready"},
			{Kind: types.FieldActual, Type: types.TypeString, Value: "123"},
		},
	}, template)
}

func TestLoadTupleInputRejectsConflictingInputs(t *testing.T) {
	_, err := LoadTupleInput("tuple.json", `job,42`)
	require.EqualError(t, err, "provide either tuple-file or tuple-spec, not both")
}

func TestParseTemplateSpecRejectsMalformedFormalField(t *testing.T) {
	_, err := ParseTemplateSpec(`job,?id`)
	require.EqualError(t, err, `parse template field 1: formal field "?id" must use ?name:type`)
}

func TestParseTupleSpecRejectsUnterminatedString(t *testing.T) {
	_, err := ParseTupleSpec(`"job,42`)
	require.EqualError(t, err, "spec contains an unterminated quoted string")
}
