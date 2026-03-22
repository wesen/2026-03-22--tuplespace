package notify

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	testpostgres "github.com/manuel/wesen/tuplespace/internal/testutil/postgres"
)

func TestNotifierReceivesDatabaseNotifications(t *testing.T) {
	db := testpostgres.Start(t)
	ctx := context.Background()

	notifier, err := New(ctx, db.URL)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, notifier.Close())
	})

	sub, err := notifier.Subscribe("jobs")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, sub.Close())
	})

	_, err = db.Pool.Exec(ctx, `SELECT pg_notify($1, '')`, ChannelName("jobs"))
	require.NoError(t, err)

	select {
	case <-sub.C():
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for notification")
	}
}
