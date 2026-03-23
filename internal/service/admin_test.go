package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/manuel/wesen/tuplespace/internal/admin"
	"github.com/manuel/wesen/tuplespace/internal/types"
)

func TestServiceStatsAndWaiters(t *testing.T) {
	svc := newTestService(t)
	template := types.Template{Fields: []types.TemplateField{
		{Kind: types.FieldActual, Type: types.TypeString, Value: "job"},
	}}

	done := make(chan error, 1)
	go func() {
		_, _, err := svc.Rd(context.Background(), "jobs", template, 2*time.Second)
		done <- err
	}()

	require.Eventually(t, func() bool {
		waiters, err := svc.Waiters(context.Background())
		require.NoError(t, err)
		return len(waiters) == 1
	}, 2*time.Second, 20*time.Millisecond)

	stats, err := svc.Stats(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, stats.WaiterCount)
	require.Equal(t, 64, stats.CandidateLimit)

	waiters, err := svc.Waiters(context.Background())
	require.NoError(t, err)
	require.Len(t, waiters, 1)
	require.Equal(t, admin.WaiterInfo{
		ID:        waiters[0].ID,
		Space:     "jobs",
		Operation: "rd",
		WaitMS:    2000,
		StartedAt: waiters[0].StartedAt,
		Template:  template,
	}, waiters[0])

	require.NoError(t, svc.Out(context.Background(), "jobs", types.Tuple{Fields: []types.TupleField{
		{Type: types.TypeString, Value: "job"},
	}}))
	require.NoError(t, <-done)

	require.Eventually(t, func() bool {
		waiters, err := svc.Waiters(context.Background())
		require.NoError(t, err)
		return len(waiters) == 0
	}, time.Second, 20*time.Millisecond)
}

func TestServiceNotifyTestReportsSubscribers(t *testing.T) {
	svc := newTestService(t)
	template := types.Template{Fields: []types.TemplateField{
		{Kind: types.FieldActual, Type: types.TypeString, Value: "job"},
	}}

	done := make(chan error, 1)
	go func() {
		_, _, err := svc.Rd(context.Background(), "jobs", template, 2*time.Second)
		done <- err
	}()

	require.Eventually(t, func() bool {
		stats, err := svc.Stats(context.Background())
		require.NoError(t, err)
		return stats.NotifierSubscribers == 1
	}, 2*time.Second, 20*time.Millisecond)

	result, err := svc.NotifyTest(context.Background(), "jobs")
	require.NoError(t, err)
	require.Equal(t, "jobs", result.Space)
	require.Equal(t, 1, result.SubscriberCount)
	require.Equal(t, 1, result.ChannelSubscriberCount)

	require.NoError(t, svc.Out(context.Background(), "jobs", types.Tuple{Fields: []types.TupleField{
		{Type: types.TypeString, Value: "job"},
	}}))
	require.NoError(t, <-done)
}
