package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/manuel/wesen/tuplespace/internal/notify"
	"github.com/manuel/wesen/tuplespace/internal/store"
	testpostgres "github.com/manuel/wesen/tuplespace/internal/testutil/postgres"
	"github.com/manuel/wesen/tuplespace/internal/types"
)

func newTestService(t *testing.T) *Service {
	t.Helper()

	db := testpostgres.Start(t)
	notifier, err := notify.New(context.Background(), db.URL)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, notifier.Close())
	})

	return New(db.Pool, store.New(), notifier, Options{
		CandidateLimit: 64,
		StartedAt:      time.Now().UTC(),
		ConfigSnapshot: RedactedConfigSnapshot(":8080", db.URL, 64, 10*time.Second),
		MigrationFiles: []string{"001_init_tuplespace.sql"},
	})
}

func TestServiceRdpIsNonDestructive(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	tuple := types.Tuple{Fields: []types.TupleField{
		{Type: types.TypeString, Value: "job"},
		{Type: types.TypeInt, Value: int64(42)},
	}}
	require.NoError(t, svc.Out(ctx, "jobs", tuple))

	template := types.Template{Fields: []types.TemplateField{
		{Kind: types.FieldActual, Type: types.TypeString, Value: "job"},
		{Kind: types.FieldFormal, Type: types.TypeInt, Name: "id"},
	}}

	firstTuple, firstBindings, ok, err := svc.Rdp(ctx, "jobs", template)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, tuple, firstTuple)
	require.Equal(t, types.Bindings{"id": int64(42)}, firstBindings)

	secondTuple, _, ok, err := svc.Rdp(ctx, "jobs", template)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, tuple, secondTuple)
}

func TestServiceInpConsumesExactlyOnce(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	require.NoError(t, svc.Out(ctx, "jobs", types.Tuple{Fields: []types.TupleField{
		{Type: types.TypeString, Value: "job"},
	}}))

	template := types.Template{Fields: []types.TemplateField{
		{Kind: types.FieldActual, Type: types.TypeString, Value: "job"},
	}}

	type result struct {
		ok  bool
		err error
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, ok, err := svc.Inp(ctx, "jobs", template)
			results <- result{ok: ok, err: err}
		}()
	}
	wg.Wait()
	close(results)

	successes := 0
	for result := range results {
		require.NoError(t, result.err)
		if result.ok {
			successes++
		}
	}
	require.Equal(t, 1, successes)
}

func TestServiceInWaitsForTupleArrival(t *testing.T) {
	svc := newTestService(t)
	template := types.Template{Fields: []types.TemplateField{
		{Kind: types.FieldActual, Type: types.TypeString, Value: "job"},
	}}

	resultCh := make(chan error, 1)
	go func() {
		_, _, err := svc.In(context.Background(), "jobs", template, 5*time.Second)
		resultCh <- err
	}()

	time.Sleep(500 * time.Millisecond)
	err := svc.Out(context.Background(), "jobs", types.Tuple{Fields: []types.TupleField{
		{Type: types.TypeString, Value: "job"},
	}})
	require.NoError(t, err)

	select {
	case err := <-resultCh:
		require.NoError(t, err)
	case <-time.After(6 * time.Second):
		t.Fatal("timed out waiting for blocking in")
	}
}
