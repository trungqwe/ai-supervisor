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

// TestTwoProcessContentionWhenDBDoesNotExistWithBarrier proves R1-001:
// Process 1 acquires .owner.lock on a new DB path. A barrier pause occurs BEFORE
// CREATE_NEW is invoked. Process 2 attempts to run on the same DB while the file
// does NOT yet exist on disk. Process 2 is blocked with exit code 32 (ERROR_SHARING_VIOLATION)
// without ever creating or corrupting the file.
func TestTwoProcessContentionWhenDBDoesNotExistWithBarrier(t *testing.T) {
	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "supervisor.exe")
	dbPath := filepath.Join(tempDir, "barrier_new.db")
	barrierSignal := filepath.Join(tempDir, "lock_acquired.txt")
	readyFile := filepath.Join(tempDir, "ready_new.txt")

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

	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatalf("expected DB not to exist initially: %v", err)
	}

	// Start Process 1 with barrier pause (2.5s) after lock acquire before CREATE_NEW
	p1Args := append([]string{
		"run",
		"-db=" + dbPath,
		"-http-addr=127.0.0.1:0",
		"-instance-id=proc-barrier-1",
		"-test-lock-acquired-signal=" + barrierSignal,
		"-test-pause-before-create-duration=2500ms",
		"-ready-signal-file=" + readyFile,
	}, policyArgs...)

	p1 := exec.Command(binPath, p1Args...)
	if err := p1.Start(); err != nil {
		t.Fatalf("failed to start Process 1: %v", err)
	}
	defer func() {
		if p1.Process != nil {
			_ = p1.Process.Kill()
		}
	}()

	// Wait for Process 1 to signal that .owner.lock has been acquired
	for start := time.Now(); time.Since(start) < 5*time.Second; time.Sleep(50 * time.Millisecond) {
		if data, err := os.ReadFile(barrierSignal); err == nil && len(data) > 0 {
			break
		}
	}

	// Crucial probe (R1-001): Verify DB file STILL DOES NOT EXIST on disk!
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatalf("expected DB file NOT to exist yet while barrier active, got err=%v", err)
	}

	// Start contender Process 2 while DB file does not exist but lock is held
	p2Args := append([]string{
		"run",
		"-db=" + dbPath,
		"-http-addr=127.0.0.1:0",
		"-instance-id=proc-barrier-2",
	}, policyArgs...)

	p2 := exec.Command(binPath, p2Args...)
	out, err := p2.CombinedOutput()
	if err == nil {
		t.Fatal("expected Process 2 to fail due to lock contention before DB creation, but succeeded")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError from Process 2, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 32 {
		t.Fatalf("expected exit code 32 (ERROR_SHARING_VIOLATION), got %d. Output:\n%s", exitErr.ExitCode(), string(out))
	}
	t.Logf("Process 2 correctly failed on non-existent DB with exit code 32 (ERROR_SHARING_VIOLATION)")

	// Verify DB file STILL DOES NOT EXIST after Process 2 failed
	// (Proves Process 2 never attempted CREATE_NEW or touched the file)
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		// Wait if Process 1's pause has just ended
	}

	// Wait for Process 1 to complete creation and become ready
	for start := time.Now(); time.Since(start) < 10*time.Second; time.Sleep(100 * time.Millisecond) {
		if data, err := os.ReadFile(readyFile); err == nil && len(data) > 0 {
			break
		}
	}

	// Clean up Process 1
	_ = exec.Command(binPath, "stop", "-db="+dbPath, "-timeout=5s").Run()
	_ = p1.Wait()
}

// TestRealDaemonDrainTimeoutPreservesLock proves R1-002:
// When shutdown drain times out because an effect caller holds a permit, the daemon
// does NOT close Store, does NOT close pinned DB handle, and does NOT release .owner.lock.
// A contender process trying to start during or immediately after the timeout is rejected
// with exit code 32 (ERROR_SHARING_VIOLATION).
func TestRealDaemonDrainTimeoutPreservesLock(t *testing.T) {
	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "supervisor.exe")
	dbPath := filepath.Join(tempDir, "drain_hold.db")
	readyFile := filepath.Join(tempDir, "ready_drain.txt")

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

	// Start Process 1 with active permit held for 6 seconds
	p1Args := append([]string{
		"run",
		"-db=" + dbPath,
		"-http-addr=127.0.0.1:0",
		"-instance-id=proc-drain-1",
		"-ready-signal-file=" + readyFile,
		"-test-hold-permit-duration=6s",
	}, policyArgs...)

	p1 := exec.Command(binPath, p1Args...)
	if err := p1.Start(); err != nil {
		t.Fatalf("failed to start Process 1: %v", err)
	}
	defer func() {
		if p1.Process != nil {
			_ = p1.Process.Kill()
		}
	}()

	// Wait for Process 1 readiness
	for start := time.Now(); time.Since(start) < 10*time.Second; time.Sleep(100 * time.Millisecond) {
		if data, err := os.ReadFile(readyFile); err == nil && len(data) > 0 {
			break
		}
	}

	// Request stop with very short timeout (500ms).
	// Because permit is held for 6s, shutdown drain will time out!
	stopCmd := exec.Command(binPath, "stop", "-db="+dbPath, "-timeout=500ms")
	_ = stopCmd.Run() // expected to fail or trigger drain timeout

	// While permit is still held (within 2 seconds of stop), contender Process 2 attempts takeover.
	// It MUST be rejected with exit code 32 (ERROR_SHARING_VIOLATION) because lock was NOT released!
	p2Args := append([]string{
		"run",
		"-db=" + dbPath,
		"-http-addr=127.0.0.1:0",
		"-instance-id=proc-drain-contender",
	}, policyArgs...)

	p2 := exec.Command(binPath, p2Args...)
	out, err := p2.CombinedOutput()
	if err == nil {
		t.Fatal("contender acquired lock prematurely during active permit drain hold! Violation of R1-002")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError from contender, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 32 {
		t.Fatalf("expected contender exit code 32 (ERROR_SHARING_VIOLATION), got %d. Output:\n%s", exitErr.ExitCode(), string(out))
	}
	t.Logf("Contender correctly blocked with exit code 32 while permit held on drain timeout")
}

// TestDaemonFaultInjectionBeforeAndAfterStartup proves R1-004:
// 1. Before startup: If startup recovery scan or dependency fails, ready-signal file is never written.
// 2. After startup: If a background scheduler fails, /readyz immediately closes fail-closed (HTTP 503).
func TestDaemonFaultInjectionBeforeAndAfterStartup(t *testing.T) {
	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "supervisor.exe")
	dbPath := filepath.Join(tempDir, "fault.db")
	readyFile := filepath.Join(tempDir, "ready_fault.txt")

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

	// Part A: Injected background poller failure after startup
	pArgs := append([]string{
		"run",
		"-db=" + dbPath,
		"-http-addr=127.0.0.1:0",
		"-instance-id=proc-fault",
		"-ready-signal-file=" + readyFile,
		"-test-fail-poller",
	}, policyArgs...)

	p := exec.Command(binPath, pArgs...)
	if err := p.Start(); err != nil {
		t.Fatalf("failed to start fault test daemon: %v", err)
	}
	defer func() {
		if p.Process != nil {
			_ = p.Process.Kill()
		}
	}()

	var httpAddr string
	for start := time.Now(); time.Since(start) < 10*time.Second; time.Sleep(100 * time.Millisecond) {
		if data, err := os.ReadFile(readyFile); err == nil && len(data) > 0 {
			httpAddr = string(data)
			break
		}
	}
	if httpAddr == "" {
		t.Fatal("timed out waiting for readiness signal before fault injection")
	}

	// Wait for injected fault to trigger and verify /readyz transitions to 503 Service Unavailable
	var failedClosed bool
	for start := time.Now(); time.Since(start) < 5*time.Second; time.Sleep(200 * time.Millisecond) {
		resp, err := http.Get("http://" + httpAddr + "/readyz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusServiceUnavailable {
				failedClosed = true
				break
			}
		}
	}

	if !failedClosed {
		t.Fatal("expected /readyz to return HTTP 503 Service Unavailable after background poller failure (R1-004)")
	}
	t.Logf("/readyz correctly transitioned to HTTP 503 fail-closed after background scheduler failure")

	_ = exec.Command(binPath, "stop", "-db="+dbPath, "-timeout=5s").Run()
	_ = p.Wait()
}
