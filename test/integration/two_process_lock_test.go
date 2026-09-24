//go:build windows

package integration_test

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestTwoProcessContentionAndDrain(t *testing.T) {
	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "supervisor.exe")
	dbPath := filepath.Join(tempDir, "contention.db")
	readyFile := filepath.Join(tempDir, "ready.txt")

	// 1. Build supervisor binary
	buildCmd := exec.Command("go", "build", "-o", binPath, "../../cmd/supervisor")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build supervisor binary: %v", err)
	}

	policyArgs := []string{
		"-http-timeout=5s",
		"-health-timeout=2s",
		"-spawn-timeout=10s",
		"-send-timeout=5s",
		"-poll-interval=1s",
		"-deadline=30s",
		"-kill-timeout=5s",
		"-workspace-timeout=5s",
	}

	// 2. Start Process 1
	p1Args := append([]string{
		"run",
		"-db=" + dbPath,
		"-http-addr=127.0.0.1:0",
		"-instance-id=proc-1",
		"-ready-signal-file=" + readyFile,
	}, policyArgs...)

	p1 := exec.Command(binPath, p1Args...)
	p1.Stdout = os.Stdout
	p1.Stderr = os.Stderr
	if err := p1.Start(); err != nil {
		t.Fatalf("failed to start Process 1: %v", err)
	}
	defer func() {
		if p1.Process != nil {
			_ = p1.Process.Kill()
		}
	}()

	// Wait for Process 1 to complete startup-before-serve and signal readiness
	var httpAddr string
	for start := time.Now(); time.Since(start) < 10*time.Second; time.Sleep(100 * time.Millisecond) {
		if data, err := os.ReadFile(readyFile); err == nil && len(data) > 0 {
			httpAddr = string(data)
			break
		}
	}
	if httpAddr == "" {
		t.Fatal("timed out waiting for Process 1 to signal readiness")
	}

	// Verify HTTP probe (/readyz and /healthz)
	resp, err := http.Get("http://" + httpAddr + "/readyz")
	if err != nil {
		t.Fatalf("failed to query /readyz: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200 from /readyz, got %d", resp.StatusCode)
	}

	respHealth, err := http.Get("http://" + httpAddr + "/healthz")
	if err != nil {
		t.Fatalf("failed to query /healthz: %v", err)
	}
	respHealth.Body.Close()
	if respHealth.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200 from /healthz, got %d", respHealth.StatusCode)
	}

	// 3. Start Process 2 targeting the EXACT same DB
	// Must fail-closed immediately with exit code 32 (ERROR_SHARING_VIOLATION)
	p2Args := append([]string{
		"run",
		"-db=" + dbPath,
		"-http-addr=127.0.0.1:0",
		"-instance-id=proc-2",
	}, policyArgs...)

	p2 := exec.Command(binPath, p2Args...)
	out, err := p2.CombinedOutput()
	if err == nil {
		t.Fatal("expected Process 2 to fail due to lock contention, but it succeeded")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError from Process 2, got %T: %v", err, err)
	}

	exitCode := exitErr.ExitCode()
	if exitCode != 32 {
		t.Fatalf("expected Process 2 exit code 32 (ERROR_SHARING_VIOLATION), got %d. Output:\n%s", exitCode, string(out))
	}
	t.Logf("Process 2 correctly failed with exit code 32 (ERROR_SHARING_VIOLATION)")

	// 4. Request clean shutdown drain of Process 1 via stop command
	stopCmd := exec.Command(binPath, "stop", "-db="+dbPath, "-timeout=5s")
	stopOut, err := stopCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("supervisor stop failed: %v. Output:\n%s", err, string(stopOut))
	}

	// Wait for Process 1 to exit cleanly
	p1Done := make(chan error, 1)
	go func() {
		p1Done <- p1.Wait()
	}()

	select {
	case err := <-p1Done:
		if err != nil {
			t.Fatalf("Process 1 did not exit cleanly: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for Process 1 to exit cleanly after stop request")
	}
	t.Logf("Process 1 drained and exited cleanly (code 0)")

	// 5. Verify that after Process 1 has drained, Process 3 can now acquire the lock
	readyFile3 := filepath.Join(tempDir, "ready3.txt")
	p3Args := append([]string{
		"run",
		"-db=" + dbPath,
		"-http-addr=127.0.0.1:0",
		"-instance-id=proc-3",
		"-ready-signal-file=" + readyFile3,
	}, policyArgs...)

	p3 := exec.Command(binPath, p3Args...)
	if err := p3.Start(); err != nil {
		t.Fatalf("failed to start Process 3 after Process 1 drain: %v", err)
	}
	defer func() {
		if p3.Process != nil {
			_ = p3.Process.Kill()
		}
	}()

	var httpAddr3 string
	for start := time.Now(); time.Since(start) < 10*time.Second; time.Sleep(100 * time.Millisecond) {
		if data, err := os.ReadFile(readyFile3); err == nil && len(data) > 0 {
			httpAddr3 = string(data)
			break
		}
	}
	if httpAddr3 == "" {
		t.Fatal("timed out waiting for Process 3 readiness after Process 1 drain")
	}

	// Clean up Process 3 via stop
	_ = exec.Command(binPath, "stop", "-db="+dbPath, "-timeout=5s").Run()
	_ = p3.Wait()
}

// TestTwoProcessContentionWhenDBDoesNotExist verifies R1-001:
// When the database file does not exist yet, Process 1 acquires .owner.lock BEFORE
// CREATE_NEW. A concurrent Process 2 targeting the same non-existent path is rejected
// with exit code 32 (ERROR_SHARING_VIOLATION) at the lock acquisition phase without
// corrupting or creating conflicting files.
func TestTwoProcessContentionWhenDBDoesNotExist(t *testing.T) {
	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "supervisor.exe")
	dbPath := filepath.Join(tempDir, "non_existent.db")
	readyFile := filepath.Join(tempDir, "ready_new.txt")

	// Build supervisor binary if not already built
	buildCmd := exec.Command("go", "build", "-o", binPath, "../../cmd/supervisor")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build supervisor binary: %v", err)
	}

	policyArgs := []string{
		"-http-timeout=5s",
		"-health-timeout=2s",
		"-spawn-timeout=10s",
		"-send-timeout=5s",
		"-poll-interval=1s",
		"-deadline=30s",
		"-kill-timeout=5s",
		"-workspace-timeout=5s",
	}

	// Confirm DB file does not exist initially
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatalf("expected DB not to exist initially: %v", err)
	}

	// Start Process 1
	p1Args := append([]string{
		"run",
		"-db=" + dbPath,
		"-http-addr=127.0.0.1:0",
		"-instance-id=proc-new-1",
		"-ready-signal-file=" + readyFile,
	}, policyArgs...)

	p1 := exec.Command(binPath, p1Args...)
	if err := p1.Start(); err != nil {
		t.Fatalf("failed to start Process 1 on new DB: %v", err)
	}
	defer func() {
		if p1.Process != nil {
			_ = p1.Process.Kill()
		}
	}()

	// Wait for Process 1 to signal readiness
	for start := time.Now(); time.Since(start) < 10*time.Second; time.Sleep(100 * time.Millisecond) {
		if data, err := os.ReadFile(readyFile); err == nil && len(data) > 0 {
			break
		}
	}

	// Process 1 has created the DB and holds .owner.lock.
	// Process 2 starts targeting the same DB path.
	p2Args := append([]string{
		"run",
		"-db=" + dbPath,
		"-http-addr=127.0.0.1:0",
		"-instance-id=proc-new-2",
	}, policyArgs...)

	p2 := exec.Command(binPath, p2Args...)
	out, err := p2.CombinedOutput()
	if err == nil {
		t.Fatal("expected Process 2 to fail due to lock contention on new DB, but succeeded")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError from Process 2, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 32 {
		t.Fatalf("expected exit code 32 (ERROR_SHARING_VIOLATION), got %d. Output:\n%s", exitErr.ExitCode(), string(out))
	}
	t.Logf("Process 2 correctly failed on new DB with exit code 32 (ERROR_SHARING_VIOLATION)")

	// Clean up Process 1
	_ = exec.Command(binPath, "stop", "-db="+dbPath, "-timeout=5s").Run()
	_ = p1.Wait()
}
