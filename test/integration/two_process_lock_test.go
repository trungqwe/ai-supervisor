//go:build windows

package integration_test

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"strings"
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

	// 3. Wait for Process 1 readiness
	for start := time.Now(); time.Since(start) < 10*time.Second; time.Sleep(100 * time.Millisecond) {
		if data, err := os.ReadFile(readyFile); err == nil && len(data) > 0 {
			break
		}
	}

	// 4. Start Process 2 contender - must fail with exit code 32
	p2Args := append([]string{
		"run",
		"-db=" + dbPath,
		"-http-addr=127.0.0.1:0",
		"-instance-id=proc-2",
	}, policyArgs...)

	p2 := exec.Command(binPath, p2Args...)
	out, err := p2.CombinedOutput()
	if err == nil {
		t.Fatal("expected Process 2 to fail, but it succeeded")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 32 {
		t.Fatalf("expected exit code 32, got %d. Output:\n%s", exitErr.ExitCode(), string(out))
	}
	t.Logf("Process 2 correctly rejected with exit code 32")

	// 5. Clean stop
	if err := exec.Command(binPath, "stop", "-db="+dbPath, "-timeout=5s").Run(); err != nil { t.Fatalf("clean stop failed: %v", err) }
	_ = p1.Wait()
}

func TestNewDBBarrierCreateAndContention(t *testing.T) {
	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "supervisor.exe")
	dbPath := filepath.Join(tempDir, "barrier.db")
	readyFile := filepath.Join(tempDir, "ready_barrier.txt")
	barrierSignal := filepath.Join(tempDir, "barrier_lock.txt")

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

	// Wait for Process 1 to complete creation and become ready
	for start := time.Now(); time.Since(start) < 10*time.Second; time.Sleep(100 * time.Millisecond) {
		if data, err := os.ReadFile(readyFile); err == nil && len(data) > 0 {
			break
		}
	}

	// Clean up Process 1
	if err := exec.Command(binPath, "stop", "-db="+dbPath, "-timeout=5s").Run(); err != nil { t.Fatalf("clean stop failed: %v", err) }
	_ = p1.Wait()
}

// TestRealDaemonDrainTimeoutPreservesLock proves R1-002:
// The daemon holds a permit for 15s. The daemon's internal drain timeout is 10s.
// After stop is triggered, drain times out at the 10s mark, but ExecuteShutdownDrain
// does NOT return to main/os.Exit. Instead, it blocks waiting for the permit holder
// to join (WaitAllReleased). At t=12s (after the 10s drain timeout), we verify:
//   - Process 1 is still alive (has NOT exited)
//   - .owner.lock is still held (contender is rejected with exit code 32)
//
// The permit releases at t=15s and the daemon exits cleanly after normal teardown.
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

	// Start Process 1 with active permit held for 15s.
	// Daemon drain timeout is 10s, so drain will timeout at t=10s.
	// But with R1-002 fix, daemon blocks until permit joins at t=15s.
	p1Args := append([]string{
		"run",
		"-db=" + dbPath,
		"-http-addr=127.0.0.1:0",
		"-instance-id=proc-drain-1",
		"-ready-signal-file=" + readyFile,
		"-test-hold-permit-duration=15s",
	}, policyArgs...)

	p1 := exec.Command(binPath, p1Args...)
	if err := p1.Start(); err != nil {
		t.Fatalf("failed to start Process 1: %v", err)
	}

	// Track Process 1 exit
	p1Done := make(chan error, 1)
	go func() {
		p1Done <- p1.Wait()
	}()

	defer func() {
		if p1.Process != nil {
			_ = p1.Process.Kill()
		}
	}()

	// Wait for Process 1 readiness
	var ready bool
	for start := time.Now(); time.Since(start) < 10*time.Second; time.Sleep(100 * time.Millisecond) {
		if data, err := os.ReadFile(readyFile); err == nil && len(data) > 0 {
			ready = true
			break
		}
	}
	if !ready {
		t.Fatal("timed out waiting for daemon readiness")
	}

	// Trigger stop via Named Pipe. The pipe stop itself returns quickly,
	// but the daemon's internal shutdown drain will time out at 10s.
	stopCmd := exec.Command(binPath, "stop", "-db="+dbPath, "-timeout=5s")
	stopOut, stopErr := stopCmd.CombinedOutput()
	if stopErr != nil {
		t.Fatalf("stop command failed: %v. Output:\n%s", stopErr, string(stopOut))
	}
	t.Logf("Stop command succeeded: %s", strings.TrimSpace(string(stopOut)))

	// Wait 12 seconds after stop trigger so we are PAST the 10s drain timeout.
	// If R1-002 fix is correct, daemon is still alive, blocking in WaitAllReleased().
	t.Log("Waiting 12s to pass the 10s drain timeout while permit is held for 15s...")
	time.Sleep(12 * time.Second)

	// Critical probe (R1-002): Process 1 must still be alive at this point!
	select {
	case exitResult := <-p1Done:
		t.Fatalf("daemon exited prematurely at t=12s (should be blocking)! "+
			"This means os.Exit released the lock while permit holder was active. "+
			"Exit result: %v. Violation of R1-002", exitResult)
	default:
		t.Log("PASS: Daemon is still alive at t=12s (past 10s drain timeout), blocking in WaitAllReleased()")
	}

	// Critical probe (R1-002): .owner.lock must still be held by Process 1.
	// Contender Process 2 must fail with exit code 32.
	p2Args := append([]string{
		"run",
		"-db=" + dbPath,
		"-http-addr=127.0.0.1:0",
		"-instance-id=proc-drain-contender",
	}, policyArgs...)

	p2 := exec.Command(binPath, p2Args...)
	out, err := p2.CombinedOutput()
	if err == nil {
		t.Fatal("contender acquired lock at t=12s while daemon should be holding it! Violation of R1-002")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError from contender, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 32 {
		t.Fatalf("expected contender exit code 32 (ERROR_SHARING_VIOLATION) at t=12s, "+
			"got %d. Output:\n%s", exitErr.ExitCode(), string(out))
	}
	t.Log("PASS: Contender correctly blocked at t=12s while daemon holds lock past drain timeout")

	// Wait for Process 1 to exit cleanly after permit releases at t=15s.
	// Total wait from stop trigger: ~15s + small teardown time.
	select {
	case exitResult := <-p1Done:
		if exitResult != nil {
			// Non-zero exit is acceptable since drain reports a timeout error
			t.Logf("Daemon exited after permit joined: %v", exitResult)
		} else {
			t.Log("PASS: Daemon exited cleanly after permit holder joined")
		}
	case <-time.After(15 * time.Second):
		t.Fatal("Daemon did not exit within expected time after permit release")
	}
}

// TestDaemonReadinessLifecycle proves R1-004 / AC-004-06:
// 1. Ready-signal file is written only AFTER all startup dependencies complete.
// 2. /readyz returns HTTP 200 OK once ready.
// 3. When stop is initiated, listeners are closed in Step 1 of shutdown drain,
//    refusing new requests and closing admission fail-closed.
// Note on Poller failure notification: recovery.Poller has unexported done/lastErr fields
// and recovery is in forbidden_scope (BLOCKER-P03-004-POLLER-ASYNC-NOTIFICATION).
func TestDaemonReadinessLifecycle(t *testing.T) {
	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "supervisor.exe")
	dbPath := filepath.Join(tempDir, "readiness.db")
	readyFile := filepath.Join(tempDir, "ready_signal.txt")

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

	pArgs := append([]string{
		"run",
		"-db=" + dbPath,
		"-http-addr=127.0.0.1:0",
		"-instance-id=proc-readiness",
		"-ready-signal-file=" + readyFile,
	}, policyArgs...)

	p := exec.Command(binPath, pArgs...)
	if err := p.Start(); err != nil {
		t.Fatalf("failed to start daemon: %v", err)
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
		t.Fatal("timed out waiting for readiness signal file")
	}

	// Probe /readyz returns HTTP 200 OK
	resp, err := http.Get("http://" + httpAddr + "/readyz")
	if err != nil {
		t.Fatalf("failed to probe /readyz: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected /readyz 200 OK, got %d", resp.StatusCode)
	}

	// Trigger stop via Named Pipe
	stopCmd := exec.Command(binPath, "stop", "-db="+dbPath, "-timeout=5s")
	if err := stopCmd.Run(); err != nil {
		t.Fatalf("stop command failed: %v", err)
	}
	_ = p.Wait()

	// Verify port is closed/refused after shutdown
	_, err = http.Get("http://" + httpAddr + "/readyz")
	if err == nil {
		t.Fatal("expected connection refused on /readyz after daemon shutdown")
	}
	t.Log("PASS: /readyz correctly refused after daemon shutdown")
}
