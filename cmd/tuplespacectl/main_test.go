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
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
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

func TestCLIUsesEnvDefaultsForServerURLAndSpace(t *testing.T) {
	db := testpostgres.Start(t)
	serverBin := buildBinary(t, "tuplespaced", "./cmd/tuplespaced")
	cliBin := buildBinary(t, "tuplespacectl", "./cmd/tuplespacectl")
	serverURL, stop := startServerProcess(t, serverBin, db.URL)
	defer stop()

	env := []string{
		"TUPLESPACECTL_SERVER_URL=" + serverURL,
		"TUPLESPACECTL_SPACE=jobs",
	}

	runCLIWithEnv(t, cliBin, env, "admin", "health", "--output", "json")
	runCLIWithEnv(t, cliBin, env, "tuple", "out", "--tuple-spec", `job,77,false`, "--output", "json")
	output := runCLIWithEnv(t, cliBin, env, "tuple", "rd", "--template-spec", `job,?id:int,?ready:bool`, "--output", "json")
	require.Contains(t, output, `"ok": true`)
	require.Contains(t, output, `"id": 77`)
	require.Contains(t, output, `"ready": false`)
}

func TestCLIOutAcceptsMultiplePositionalTupleSpecs(t *testing.T) {
	db := testpostgres.Start(t)
	serverBin := buildBinary(t, "tuplespaced", "./cmd/tuplespaced")
	cliBin := buildBinary(t, "tuplespacectl", "./cmd/tuplespacectl")
	serverURL, stop := startServerProcess(t, serverBin, db.URL)
	defer stop()

	output := runCLI(t, cliBin, serverURL, "tuple", "out", "--space", "jobs", "--output", "json", `job,1,true`, `job,2,false`)
	require.Contains(t, output, `"index": 0`)
	require.Contains(t, output, `"index": 1`)

	first := runCLI(t, cliBin, serverURL, "tuple", "in", "--space", "jobs", "--template-spec", `job,?id:int,?ready:bool`, "--output", "json")
	require.Contains(t, first, `"id": 1`)
	require.Contains(t, first, `"ready": true`)

	second := runCLI(t, cliBin, serverURL, "tuple", "in", "--space", "jobs", "--template-spec", `job,?id:int,?ready:bool`, "--output", "json")
	require.Contains(t, second, `"id": 2`)
	require.Contains(t, second, `"ready": false`)
}

func TestCLIRdAcceptsMultiplePositionalTemplateSpecs(t *testing.T) {
	db := testpostgres.Start(t)
	serverBin := buildBinary(t, "tuplespaced", "./cmd/tuplespaced")
	cliBin := buildBinary(t, "tuplespacectl", "./cmd/tuplespacectl")
	serverURL, stop := startServerProcess(t, serverBin, db.URL)
	defer stop()

	runCLI(t, cliBin, serverURL, "tuple", "out", "--space", "jobs", "--output", "json", `job,1,true`, `worker,2,false`)

	output := runCLI(t, cliBin, serverURL, "tuple", "rd", "--space", "jobs", "--output", "json", `job,?id:int,?ready:bool`, `worker,?id:int,?ready:bool`)
	require.Contains(t, output, `"index": 0`)
	require.Contains(t, output, `"index": 1`)
	require.Contains(t, output, `"id": 1`)
	require.Contains(t, output, `"ready": true`)
	require.Contains(t, output, `"id": 2`)
	require.Contains(t, output, `"ready": false`)
}

func TestCLIInAcceptsMultiplePositionalTemplateSpecs(t *testing.T) {
	db := testpostgres.Start(t)
	serverBin := buildBinary(t, "tuplespaced", "./cmd/tuplespaced")
	cliBin := buildBinary(t, "tuplespacectl", "./cmd/tuplespacectl")
	serverURL, stop := startServerProcess(t, serverBin, db.URL)
	defer stop()

	runCLI(t, cliBin, serverURL, "tuple", "out", "--space", "jobs", "--output", "json", `job,1,true`, `worker,2,false`)

	output := runCLI(t, cliBin, serverURL, "tuple", "in", "--space", "jobs", "--output", "json", `job,?id:int,?ready:bool`, `worker,?id:int,?ready:bool`)
	require.Contains(t, output, `"index": 0`)
	require.Contains(t, output, `"index": 1`)
	require.Contains(t, output, `"id": 1`)
	require.Contains(t, output, `"ready": true`)
	require.Contains(t, output, `"id": 2`)
	require.Contains(t, output, `"ready": false`)

	missingJob := runCLIExpectError(t, cliBin, serverURL, "tuple", "rd", "--space", "jobs", "--template-spec", `job,?id:int,?ready:bool`, "--output", "json")
	require.Contains(t, missingJob, "Error: not_found: tuple not found")

	missingWorker := runCLIExpectError(t, cliBin, serverURL, "tuple", "rd", "--space", "jobs", "--template-spec", `worker,?id:int,?ready:bool`, "--output", "json")
	require.Contains(t, missingWorker, "Error: not_found: tuple not found")
}

func TestCLIAdminReadOnlyCommands(t *testing.T) {
	db := testpostgres.Start(t)
	serverBin := buildBinary(t, "tuplespaced", "./cmd/tuplespaced")
	cliBin := buildBinary(t, "tuplespacectl", "./cmd/tuplespacectl")
	serverURL, stop := startServerProcess(t, serverBin, db.URL)
	defer stop()

	runCLI(t, cliBin, serverURL, "tuple", "out", "--space", "jobs", "--output", "json", `job,1,true`, `job,2,false`)
	runCLI(t, cliBin, serverURL, "tuple", "out", "--space", "workers", "--output", "json", `worker,3,true`)

	spaces := runCLI(t, cliBin, serverURL, "admin", "spaces", "--output", "json")
	require.Contains(t, spaces, `"space": "jobs"`)
	require.Contains(t, spaces, `"space": "workers"`)

	dump := runCLI(t, cliBin, serverURL, "admin", "dump", "--space", "jobs", "--output", "json")
	require.Contains(t, dump, `"space": "jobs"`)
	require.Contains(t, dump, `"id":`)

	stats := runCLI(t, cliBin, serverURL, "admin", "stats", "--output", "json")
	require.Contains(t, stats, `"tuple_count": 3`)
	require.Contains(t, stats, `"space_count": 2`)

	config := runCLI(t, cliBin, serverURL, "admin", "config", "--output", "json")
	require.Contains(t, config, `"database_url":`)
	require.Contains(t, config, `"candidate_limit": 64`)

	schema := runCLI(t, cliBin, serverURL, "admin", "schema", "--output", "json")
	require.Contains(t, schema, `"migration_files":`)
	require.Contains(t, schema, `"tuples_space_arity_id_idx"`)
}

func TestCLIAdminTupleAndFilteredCommands(t *testing.T) {
	db := testpostgres.Start(t)
	serverBin := buildBinary(t, "tuplespaced", "./cmd/tuplespaced")
	cliBin := buildBinary(t, "tuplespacectl", "./cmd/tuplespacectl")
	serverURL, stop := startServerProcess(t, serverBin, db.URL)
	defer stop()

	runCLI(t, cliBin, serverURL, "tuple", "out", "--space", "jobs", "--output", "json", `job,1,true`, `job,2,false`)
	runCLI(t, cliBin, serverURL, "tuple", "out", "--space", "workers", "--output", "json", `worker,3,true`)

	jobTupleID := lookupTupleID(t, db.Pool, "jobs")
	workerTupleID := lookupTupleID(t, db.Pool, "workers")

	getOutput := runCLI(t, cliBin, serverURL, "admin", "tuple", "get", "--tuple-id", fmt.Sprintf("%d", jobTupleID), "--output", "json")
	require.Contains(t, getOutput, fmt.Sprintf(`"id": %d`, jobTupleID))
	require.Contains(t, getOutput, `"space": "jobs"`)

	peekOutput := runCLI(t, cliBin, serverURL, "admin", "peek", "--space", "workers", "--output", "json")
	require.Contains(t, peekOutput, `"space": "workers"`)
	require.Contains(t, peekOutput, `"worker"`)

	exportOutput := runCLI(t, cliBin, serverURL, "admin", "export", "--space", "jobs", "--output", "json")
	require.Contains(t, exportOutput, `"space": "jobs"`)
	require.Contains(t, exportOutput, `"job"`)

	deleteOutput := runCLI(t, cliBin, serverURL, "admin", "tuple", "delete", "--tuple-id", fmt.Sprintf("%d", workerTupleID), "--output", "json")
	require.Contains(t, deleteOutput, fmt.Sprintf(`"tuple_id": %d`, workerTupleID))
	require.Contains(t, deleteOutput, `"deleted": true`)

	missing := runCLIExpectError(t, cliBin, serverURL, "admin", "tuple", "get", "--tuple-id", fmt.Sprintf("%d", workerTupleID), "--output", "json")
	require.Contains(t, missing, "Error: not_found: tuple not found")
}

func TestCLIAdminPurgeAndNotifyTest(t *testing.T) {
	db := testpostgres.Start(t)
	serverBin := buildBinary(t, "tuplespaced", "./cmd/tuplespaced")
	cliBin := buildBinary(t, "tuplespacectl", "./cmd/tuplespacectl")
	serverURL, stop := startServerProcess(t, serverBin, db.URL)
	defer stop()

	runCLI(t, cliBin, serverURL, "tuple", "out", "--space", "jobs", "--output", "json", `job,1,true`, `job,2,false`)
	runCLI(t, cliBin, serverURL, "tuple", "out", "--space", "workers", "--output", "json", `worker,3,true`)

	missingConfirm := runCLIExpectError(t, cliBin, serverURL, "admin", "purge", "--space", "jobs", "--output", "json")
	require.Contains(t, missingConfirm, "Error: confirm_required: confirm flag required")

	purgeOutput := runCLI(t, cliBin, serverURL, "admin", "purge", "--space", "jobs", "--confirm", "--output", "json")
	require.Contains(t, purgeOutput, `"deleted_count": 2`)

	workers := runCLI(t, cliBin, serverURL, "admin", "dump", "--space", "workers", "--output", "json")
	require.Contains(t, workers, `"space": "workers"`)

	readerDone := make(chan string, 1)
	readerErr := make(chan error, 1)
	go func() {
		output, err := runCLIResult(cliBin, serverURL, "tuple", "rd", "--space", "jobs", "--template-spec", `job,?id:int,?ready:bool`, "--wait-ms", "3000", "--output", "json")
		readerDone <- output
		readerErr <- err
	}()

	require.Eventually(t, func() bool {
		output := runCLI(t, cliBin, serverURL, "admin", "waiters", "--output", "json")
		return strings.Contains(output, `"space": "jobs"`)
	}, 2*time.Second, 50*time.Millisecond)

	notifyOutput := runCLI(t, cliBin, serverURL, "admin", "notify-test", "--space", "jobs", "--output", "json")
	require.Contains(t, notifyOutput, `"space": "jobs"`)
	require.Contains(t, notifyOutput, `"subscriber_count": 1`)
	require.Contains(t, notifyOutput, `"channel_subscriber_count": 1`)

	runCLI(t, cliBin, serverURL, "tuple", "out", "--space", "jobs", "--output", "json", `job,9,true`)
	require.NoError(t, <-readerErr)
	require.Contains(t, <-readerDone, `"id": 9`)
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

	output, err := runCLIResult(cliBin, serverURL, args...)
	require.NoError(t, err, string(output))
	return output
}

func runCLIResult(cliBin string, serverURL string, args ...string) (string, error) {
	allArgs := append([]string{}, args...)
	allArgs = append(allArgs, "--server-url", serverURL)
	cmd := exec.Command(cliBin, allArgs...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func runCLIExpectError(t *testing.T, cliBin string, serverURL string, args ...string) string {
	t.Helper()

	allArgs := append([]string{}, args...)
	allArgs = append(allArgs, "--server-url", serverURL)
	cmd := exec.Command(cliBin, allArgs...)
	output, err := cmd.CombinedOutput()
	require.Error(t, err, string(output))
	return string(output)
}

func runCLIWithEnv(t *testing.T, cliBin string, env []string, args ...string) string {
	t.Helper()

	cmd := exec.Command(cliBin, args...)
	cmd.Env = append(os.Environ(), env...)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
	return string(output)
}

func lookupTupleID(t *testing.T, pool *pgxpool.Pool, space string) int64 {
	t.Helper()

	var tupleID int64
	err := pool.QueryRow(context.Background(), `SELECT id FROM tuples WHERE space = $1 ORDER BY id LIMIT 1`, space).Scan(&tupleID)
	require.NoError(t, err)
	return tupleID
}
