package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	testpostgres "github.com/manuel/wesen/tuplespace/internal/testutil/postgres"
)

func TestCLIHealthAndTupleRoundTrip(t *testing.T) {
	db := testpostgres.Start(t)
	serverBin := buildBinary(t, "tuplespaced", "./cmd/tuplespaced")
	cliBin := buildBinary(t, "tuplespacectl", "./cmd/tuplespacectl")
	serverURL, stop := startServerProcess(t, serverBin, db.URL)
	defer stop()

	tempDir := t.TempDir()
	tupleFile := filepath.Join(tempDir, "tuple.json")
	templateFile := filepath.Join(tempDir, "template.json")
	require.NoError(t, os.WriteFile(tupleFile, []byte(`{"fields":[{"type":"string","value":"job"},{"type":"int","value":42}]}`), 0o644))
	require.NoError(t, os.WriteFile(templateFile, []byte(`{"fields":[{"kind":"actual","type":"string","value":"job"},{"kind":"formal","type":"int","name":"id"}]}`), 0o644))

	runCLI(t, cliBin, serverURL, "admin", "health", "--output", "json")
	runCLI(t, cliBin, serverURL, "tuple", "out", "--space", "jobs", "--tuple-file", tupleFile, "--output", "json")
	output := runCLI(t, cliBin, serverURL, "tuple", "rd", "--space", "jobs", "--template-json-file", templateFile, "--output", "json")
	require.Contains(t, output, `"ok": true`)
	require.Contains(t, output, `"space": "jobs"`)
}

func TestCLITupleRoundTripWithDSL(t *testing.T) {
	db := testpostgres.Start(t)
	serverBin := buildBinary(t, "tuplespaced", "./cmd/tuplespaced")
	cliBin := buildBinary(t, "tuplespacectl", "./cmd/tuplespacectl")
	serverURL, stop := startServerProcess(t, serverBin, db.URL)
	defer stop()

	runCLI(t, cliBin, serverURL, "tuple", "out", "--space", "jobs", "--tuple-spec", `job,42,true`, "--output", "json")
	output := runCLI(t, cliBin, serverURL, "tuple", "rd", "--space", "jobs", "--template-spec", `job,?id:int,?ready:bool`, "--output", "json")
	require.Contains(t, output, `"ok": true`)
	require.Contains(t, output, `"id": 42`)
	require.Contains(t, output, `"ready": true`)
}

func startServerProcess(t *testing.T, serverBin string, databaseURL string) (string, func()) {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))

	port := reservePort(t)
	cmd := exec.Command(serverBin)
	cmd.Dir = projectRoot
	cmd.Env = append(os.Environ(),
		"TUPLESPACE_DATABASE_URL="+databaseURL,
		"TUPLESPACE_HTTP_LISTEN_ADDR="+port,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Start())

	serverURL := "http://" + port
	waitForHealth(t, serverURL)
	return serverURL, func() {
		if cmd.Process != nil {
			_ = cmd.Process.Signal(os.Interrupt)
		}
		done := make(chan struct{})
		go func() {
			_ = cmd.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		}
	}
}

func buildBinary(t *testing.T, name, pkg string) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))

	outputPath := filepath.Join(t.TempDir(), name)
	cmd := exec.Command("go", "build", "-o", outputPath, pkg)
	cmd.Dir = projectRoot
	buildOutput, err := cmd.CombinedOutput()
	require.NoError(t, err, string(buildOutput))
	return outputPath
}

func reservePort(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	return fmt.Sprintf("127.0.0.1:%d", listener.Addr().(*net.TCPAddr).Port)
}

func waitForHealth(t *testing.T, serverURL string) {
	t.Helper()

	deadline := time.Now().Add(15 * time.Second)
	client := &http.Client{Timeout: time.Second}
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, serverURL+"/healthz", nil)
		require.NoError(t, err)
		res, err := client.Do(req)
		if err == nil && res.StatusCode == http.StatusOK {
			_ = res.Body.Close()
			return
		}
		if res != nil {
			_ = res.Body.Close()
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", serverURL)
}

func runCLI(t *testing.T, cliBin string, serverURL string, args ...string) string {
	t.Helper()

	allArgs := append([]string{}, args...)
	allArgs = append(allArgs, "--server-url", serverURL)
	cmd := exec.Command(cliBin, allArgs...)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
	return string(output)
}
