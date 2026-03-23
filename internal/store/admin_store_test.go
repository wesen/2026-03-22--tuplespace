package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/manuel/wesen/tuplespace/internal/admin"
	"github.com/manuel/wesen/tuplespace/internal/testutil/postgres"
	"github.com/manuel/wesen/tuplespace/internal/types"
)

func TestTupleStoreListSpacesAndTuples(t *testing.T) {
	db := postgres.Start(t)
	ctx := context.Background()
	store := New()

	insertTuple := func(space string, tuple types.Tuple) int64 {
		tx, err := db.Pool.Begin(ctx)
		require.NoError(t, err)
		tupleID, err := store.InsertTuple(ctx, tx, space, tuple)
		require.NoError(t, err)
		require.NoError(t, tx.Commit(ctx))
		return tupleID
	}

	jobID := insertTuple("jobs", types.Tuple{Fields: []types.TupleField{{Type: types.TypeString, Value: "job"}}})
	workerID := insertTuple("workers", types.Tuple{Fields: []types.TupleField{{Type: types.TypeString, Value: "worker"}}})

	spaces, err := store.ListSpaces(ctx, db.Pool)
	require.NoError(t, err)
	require.Len(t, spaces, 2)
	require.Equal(t, "jobs", spaces[0].Space)
	require.EqualValues(t, 1, spaces[0].TupleCount)
	require.Equal(t, "workers", spaces[1].Space)
	require.EqualValues(t, 1, spaces[1].TupleCount)

	tuples, err := store.ListTuples(ctx, db.Pool, admin.TupleFilter{Space: "jobs"})
	require.NoError(t, err)
	require.Len(t, tuples, 1)
	require.Equal(t, jobID, tuples[0].ID)
	require.Equal(t, "jobs", tuples[0].Space)
	require.Equal(t, 1, tuples[0].Arity)
	require.False(t, tuples[0].CreatedAt.IsZero())

	record, found, err := store.GetTupleByID(ctx, db.Pool, workerID)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, workerID, record.ID)
	require.Equal(t, "workers", record.Space)
}

func TestTupleStoreCountAndDeleteByFilter(t *testing.T) {
	db := postgres.Start(t)
	ctx := context.Background()
	store := New()

	insertedAt := time.Now().UTC().Add(-time.Hour)
	tx, err := db.Pool.Begin(ctx)
	require.NoError(t, err)
	tupleID, err := store.InsertTuple(ctx, tx, "jobs", types.Tuple{Fields: []types.TupleField{{Type: types.TypeString, Value: "job"}}})
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))

	_, err = db.Pool.Exec(ctx, `UPDATE tuples SET created_at = $1 WHERE id = $2`, insertedAt, tupleID)
	require.NoError(t, err)

	count, err := store.CountTuples(ctx, db.Pool, admin.TupleFilter{Space: "jobs"})
	require.NoError(t, err)
	require.EqualValues(t, 1, count)

	purgeTx, err := db.Pool.Begin(ctx)
	require.NoError(t, err)
	deleted, err := store.DeleteTuples(ctx, purgeTx, admin.TupleFilter{
		Space:         "jobs",
		CreatedBefore: ptr(insertedAt.Add(time.Minute)),
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, deleted)
	require.NoError(t, purgeTx.Commit(ctx))

	count, err = store.CountTuples(ctx, db.Pool, admin.TupleFilter{Space: "jobs"})
	require.NoError(t, err)
	require.Zero(t, count)
}

func ptr(t time.Time) *time.Time {
	return &t
}
