package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTimeoutForWaitMSUsesDefaultForImmediateRequests(t *testing.T) {
	require.Equal(t, defaultHTTPTimeout, TimeoutForWaitMS(0))
	require.Equal(t, defaultHTTPTimeout, TimeoutForWaitMS(1_000))
}

func TestTimeoutForWaitMSExtendsBeyondLongBlockingWaits(t *testing.T) {
	require.Equal(t, 1_005*time.Second, TimeoutForWaitMS(1_000_000))
}

func TestNewWithTimeoutFallsBackToDefault(t *testing.T) {
	client := NewWithTimeout("http://127.0.0.1:8080", 0)
	require.Equal(t, defaultHTTPTimeout, client.http.Timeout)
}

func TestNewWithTimeoutUsesExplicitTimeout(t *testing.T) {
	client := NewWithTimeout("http://127.0.0.1:8080", 42*time.Second)
	require.Equal(t, 42*time.Second, client.http.Timeout)
}
