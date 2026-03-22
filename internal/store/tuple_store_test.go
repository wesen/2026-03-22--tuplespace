package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/manuel/wesen/tuplespace/internal/testutil/postgres"
	"github.com/manuel/wesen/tuplespace/internal/types"
)

func TestTupleStoreInsertAndFindCandidates(t *testing.T) {
	db := postgres.Start(t)
	ctx := context.Background()
	store := New()

	tx, err := db.Pool.Begin(ctx)
	require.NoError(t, err)

	tuple := types.Tuple{
		Fields: []types.TupleField{
			{Type: types.TypeString, Value: "job"},
			{Type: types.TypeInt, Value: int64(42)},
			{Type: types.TypeBool, Value: true},
		},
	}

	_, err = store.InsertTuple(ctx, tx, "jobs", tuple)
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))

	template := types.Template{
		Fields: []types.TemplateField{
			{Kind: types.FieldActual, Type: types.TypeString, Value: "job"},
			{Kind: types.FieldFormal, Type: types.TypeInt, Name: "id"},
			{Kind: types.FieldActual, Type: types.TypeBool, Value: true},
		},
	}

	candidates, err := store.FindCandidates(ctx, db.Pool, "jobs", template, 64)
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.Equal(t, tuple, candidates[0].Tuple)
}

func TestTupleStoreLockCandidatesAndDelete(t *testing.T) {
	db := postgres.Start(t)
	ctx := context.Background()
	store := New()

	tx, err := db.Pool.Begin(ctx)
	require.NoError(t, err)

	tuple := types.Tuple{
		Fields: []types.TupleField{
			{Type: types.TypeString, Value: "job"},
			{Type: types.TypeInt, Value: int64(1)},
		},
	}
	tupleID, err := store.InsertTuple(ctx, tx, "jobs", tuple)
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))

	consumeTx, err := db.Pool.Begin(ctx)
	require.NoError(t, err)

	template := types.Template{
		Fields: []types.TemplateField{
			{Kind: types.FieldActual, Type: types.TypeString, Value: "job"},
			{Kind: types.FieldFormal, Type: types.TypeInt, Name: "id"},
		},
	}

	candidates, err := store.LockCandidatesForConsume(ctx, consumeTx, "jobs", template, 64)
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.Equal(t, tupleID, candidates[0].ID)

	require.NoError(t, store.DeleteTuple(ctx, consumeTx, tupleID))
	require.NoError(t, consumeTx.Commit(ctx))

	count, err := postgres.CountTuples(ctx, db.Pool)
	require.NoError(t, err)
	require.Zero(t, count)
}
