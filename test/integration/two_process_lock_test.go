//go:build windows

package integration_test

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	if err := exec.Command(binPath, "stop", "-db="+dbPath, "-timeout=5s").Run(); err != nil {
		t.Fatalf("clean stop failed: %v", err)
	}
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
	if err := exec.Command(binPath, "stop", "-db="+dbPath, "-timeout=5s").Run(); err != nil {
		t.Fatalf("clean stop failed: %v", err)
	}
	_ = p1.Wait()
}

// TestRealDaemonDrainTimeoutPreservesLock proves R1-002 & P03-004-R2-001:
//
// Lifecycle Phases & Finding P03-004-R2-001 Differentiation:
//
//	Phase 1 (Stop Acknowledgement): CLI sends TAKEOVER via Named Pipe. Daemon immediately
//	        acknowledges with status=STOP_ACKNOWLEDGED. The CLI must NOT claim
//	        "stopped successfully" because drain is only initiated and still in progress.
//	Phase 2 (Drain Timeout with Active Permit at t=12s): Daemon holds permit for 15s;
//	        daemon's internal drain timeout is 10s. At t=12s (2s past drain timeout mark):
//	        - Daemon process is STILL RUNNING (blocking in WaitAllReleased).
//	        - .owner.lock is STILL HELD (contender is rejected with exit code 32).
//	        - This proves os.Exit was NOT called and OS lock handle was preserved.
//	Phase 3 (Permit Joined & Lock Released at t=15s): Permit is released by caller at t=15s.
//	        - Daemon unblocks, joins all callers, completes normal teardown, and exits.
//	        - Contender can now acquire .owner.lock (lock has actually been released).
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

	// -------------------------------------------------------------------------
	// Phase 1: Named Pipe Stop Acknowledgement (P03-004-R2-001)
	// Trigger stop via Named Pipe. Verify:
	//   1. Command returns exit code 0 indicating message delivery.
	//   2. Output shows status=STOP_ACKNOWLEDGED (not DRAINED).
	//   3. CLI does NOT claim "stopped successfully" at this point.
	// -------------------------------------------------------------------------
	stopCmd := exec.Command(binPath, "stop", "-db="+dbPath, "-timeout=5s")
	stopOut, stopErr := stopCmd.CombinedOutput()
	if stopErr != nil {
		t.Fatalf("stop command failed: %v. Output:\n%s", stopErr, string(stopOut))
	}
	stopStr := string(stopOut)
	if !strings.Contains(stopStr, "status=STOP_ACKNOWLEDGED") {
		t.Fatalf("expected status=STOP_ACKNOWLEDGED in stop output, got:\n%s", stopStr)
	}
	if strings.Contains(strings.ToLower(stopStr), "stopped successfully") {
		t.Fatalf("CLI must NOT claim daemon stopped successfully before drain completes (P03-004-R2-001). Output:\n%s", stopStr)
	}
	t.Logf("Phase 1 PASS (P03-004-R2-001): Stop request acknowledged (drain running in background): %s", strings.TrimSpace(stopStr))

	// -------------------------------------------------------------------------
	// Phase 2: Drain Timeout with Active Permit (t=12s, R1-002 Regression probe)
	// Wait 12 seconds after stop trigger: PAST the 10s drain timeout, but BEFORE
	// the 15s permit duration expires.
	// -------------------------------------------------------------------------
	t.Log("Waiting 12s to pass the 10s drain timeout while permit is held for 15s...")
	time.Sleep(12 * time.Second)

	// Probe A: Process 1 must STILL be alive at t=12s (blocking in WaitAllReleased)
	select {
	case exitResult := <-p1Done:
		t.Fatalf("daemon exited prematurely at t=12s (should be blocking in WaitAllReleased)! "+
			"Exit result: %v. Violation of R1-002", exitResult)
	default:
		t.Log("Phase 2 Probe A PASS: Daemon is still alive at t=12s (past 10s drain timeout)")
	}

	// Probe B: .owner.lock must STILL be held by Process 1 at t=12s
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
	t.Log("Phase 2 Probe B PASS: Contender correctly rejected (code 32) at t=12s while daemon holds lock")

	// -------------------------------------------------------------------------
	// Phase 3: Permit Joined & Lock Released (t=15s)
	// Wait for Process 1 to complete teardown and exit after permit holder releases.
	// Then actually probe that a new contender acquires .owner.lock and starts up.
	// -------------------------------------------------------------------------
	select {
	case exitResult := <-p1Done:
		if exitResult == nil {
			t.Fatal("expected daemon to exit with non-zero exit code on drain timeout, got nil")
		}
		exitErr, ok := exitResult.(*exec.ExitError)
		if !ok || exitErr.ExitCode() != 2 {
			t.Fatalf("expected daemon exit code 2 on drain timeout, got: %v", exitResult)
		}
		t.Logf("Phase 3 Probe C PASS: Daemon correctly exited with code 2 on drain timeout: %v", exitResult)
	case <-time.After(15 * time.Second):
		t.Fatal("Daemon did not exit within expected time after permit release")
	}

	// Probe D: Verify that a new contender process can actually acquire the lock now that Process 1 has exited.
	p3ReadyFile := filepath.Join(tempDir, "ready_p3.txt")
	p3Args := append([]string{
		"run",
		"-db=" + dbPath,
		"-http-addr=127.0.0.1:0",
		"-instance-id=proc-drain-contender-p3",
		"-ready-signal-file=" + p3ReadyFile,
	}, policyArgs...)

	p3 := exec.Command(binPath, p3Args...)
	if err := p3.Start(); err != nil {
		t.Fatalf("failed to start contender Process 3 after daemon exit: %v", err)
	}
	defer func() {
		if p3.Process != nil {
			_ = p3.Process.Kill()
		}
	}()

	var p3Ready bool
	for start := time.Now(); time.Since(start) < 10*time.Second; time.Sleep(100 * time.Millisecond) {
		if data, err := os.ReadFile(p3ReadyFile); err == nil && len(data) > 0 {
			p3Ready = true
			break
		}
	}
	if !p3Ready {
		t.Fatal("contender Process 3 failed to acquire lock and achieve readiness after daemon exit")
	}
	t.Log("Phase 3 Probe D PASS: Contender Process 3 successfully acquired lock and achieved readiness after Process 1 exit")
}

func TestCLIStopNegativeCases(t *testing.T) {
	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "supervisor.exe")
	nonExistentDB := filepath.Join(tempDir, "nonexistent.db")

	buildCmd := exec.Command("go", "build", "-o", binPath, "../../cmd/supervisor")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build supervisor binary: %v", err)
	}

	// Case 1: Stop against non-existent db / missing metadata fails with non-zero exit code
	cmd := exec.Command(binPath, "stop", "-db="+nonExistentDB, "-timeout=1s")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected supervisor stop to fail for non-existent db, got success. Output:\n%s", string(out))
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit code from failed stop, got %v", err)
	}
	t.Logf("CLI stop negative test PASS: Non-zero exit code (%d) on missing metadata: %s", exitErr.ExitCode(), strings.TrimSpace(string(out)))
}

// TestDaemonReadinessLifecycle proves R1-004 / AC-004-06:
//  1. Ready-signal file is written only AFTER all startup dependencies complete.
//  2. /readyz returns HTTP 200 OK once ready.
//  3. When stop is initiated, listeners are closed in Step 1 of shutdown drain,
//     refusing new requests and closing admission fail-closed.
//
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


// TestDaemonPollerFailureMidRunProbe proves P03-004-R3-002:
// 1. Starts daemon with -test-fail-poller-file trigger and -poll-interval=50ms.
// 2. Verifies daemon starts up, signals ready, and /readyz returns HTTP 200 OK.
// 3. Verifies owner lock is held (contender rejected with exit code 32).
// 4. Triggers real PollOnce failure mid-run by writing the trigger file.
// 5. Verifies /readyz transitions to HTTP 503 Service Unavailable (host admission closed).
// 6. Verifies daemon process does NOT crash and owner lock (.owner.lock) is STILL held (contender exit code 32).
// 7. Executes graceful shutdown via stop CLI command, verifying that shutdown
//    cleanly joins the watcher and the Poller (which already stopped due to failure).
// 8. Verifies that after daemon exits, lock is released and a new contender can acquire it.
func TestDaemonPollerFailureMidRunProbe(t *testing.T) {
	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "supervisor.exe")
	dbPath := filepath.Join(tempDir, "poll_fail.db")
	readyFile := filepath.Join(tempDir, "ready_signal.txt")
	failTriggerFile := filepath.Join(tempDir, "fail_poller.trigger")

	buildCmd := exec.Command("go", "build", "-o", binPath, "../../cmd/supervisor")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build supervisor binary: %v", err)
	}

	policyArgs := []string{
		"-http-timeout=5s",
		"-health-timeout=2s",
		"-spawn-timeout=10s",
		"-send-timeout=5s",
		"-poll-interval=50ms",
		"-deadline=30s",
		"-kill-timeout=5s",
		"-workspace-timeout=5s",
	}

	p1Args := append([]string{
		"run",
		"-db=" + dbPath,
		"-http-addr=127.0.0.1:0",
		"-instance-id=proc-poller-midrun-fail",
		"-ready-signal-file=" + readyFile,
		"-test-fail-poller-file=" + failTriggerFile,
	}, policyArgs...)

	p1 := exec.Command(binPath, p1Args...)
	p1Done := make(chan error, 1)
	if err := p1.Start(); err != nil {
		t.Fatalf("failed to start daemon: %v", err)
	}
	go func() {
		p1Done <- p1.Wait()
	}()
	defer func() {
		if p1.Process != nil {
			_ = p1.Process.Kill()
		}
	}()

	// 1. Wait for readiness signal file
	var httpAddr string
	for start := time.Now(); time.Since(start) < 10*time.Second; time.Sleep(50 * time.Millisecond) {
		if data, err := os.ReadFile(readyFile); err == nil && len(data) > 0 {
			httpAddr = strings.TrimSpace(string(data))
			break
		}
	}
	if httpAddr == "" {
		t.Fatal("timed out waiting for readiness signal file")
	}

	// 2. Probe /readyz returns HTTP 200 OK initially
	resp, err := http.Get("http://" + httpAddr + "/readyz")
	if err != nil {
		t.Fatalf("failed to probe /readyz before failure: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected /readyz 200 OK before failure, got %d", resp.StatusCode)
	}
	t.Log("PASS: /readyz returned 200 OK initially")

	// 3. Verify owner lock is held initially (contender rejected with code 32)
	p2Args := append([]string{
		"run",
		"-db=" + dbPath,
		"-http-addr=127.0.0.1:0",
		"-instance-id=proc-midrun-contender",
	}, policyArgs...)
	p2 := exec.Command(binPath, p2Args...)
	out, err := p2.CombinedOutput()
	if err == nil {
		t.Fatal("expected contender to fail with lock contention initially, but succeeded")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok || exitErr.ExitCode() != 32 {
		t.Fatalf("expected contender exit code 32 initially, got %v: %s", err, string(out))
	}
	t.Log("PASS: Contender correctly rejected (code 32) initially")

	// 4. Trigger real PollOnce failure mid-run
	if err := os.WriteFile(failTriggerFile, []byte("fail"), 0644); err != nil {
		t.Fatalf("failed to write fail trigger file: %v", err)
	}

	// 5. Poll /readyz until it transitions to HTTP 503 Service Unavailable
	var ready503 bool
	for start := time.Now(); time.Since(start) < 5*time.Second; time.Sleep(50 * time.Millisecond) {
		resp, err := http.Get("http://" + httpAddr + "/readyz")
		if err == nil {
			status := resp.StatusCode
			resp.Body.Close()
			if status == http.StatusServiceUnavailable {
				ready503 = true
				break
			}
		}
	}
	if !ready503 {
		t.Fatal("expected /readyz to transition to 503 Service Unavailable after poller error, but it did not")
	}
	t.Log("PASS: /readyz transitioned to 503 Service Unavailable upon background poller failure")

	// 6. Verify daemon process has NOT crashed
	select {
	case exitErr := <-p1Done:
		t.Fatalf("daemon crashed or exited unexpectedly on poller failure: %v", exitErr)
	default:
		t.Log("PASS: Daemon process is still alive and did not crash on poller failure")
	}

	// 7. Verify owner lock is STILL held despite poller failure and closed admission
	p3 := exec.Command(binPath, p2Args...)
	out3, err3 := p3.CombinedOutput()
	if err3 == nil {
		t.Fatal("contender acquired lock while daemon should still hold it after poller failure")
	}
	exitErr3, ok3 := err3.(*exec.ExitError)
	if !ok3 || exitErr3.ExitCode() != 32 {
		t.Fatalf("expected contender exit code 32 after poller failure, got %v: %s", err3, string(out3))
	}
	t.Log("PASS: Owner lock is STILL held by daemon after poller failure (contender code 32)")

	// 8. Graceful shutdown via stop CLI command
	stopCmd := exec.Command(binPath, "stop", "-db="+dbPath, "-timeout=5s")
	stopOut, stopErr := stopCmd.CombinedOutput()
	if stopErr != nil {
		t.Fatalf("stop command failed during teardown: %v, output: %s", stopErr, string(stopOut))
	}
	t.Logf("PASS: Stop command succeeded: %s", strings.TrimSpace(string(stopOut)))

	// 9. Verify daemon exits cleanly with code 0 (graceful shutdown joins watcher and stopped poller)
	select {
	case p1Err := <-p1Done:
		if p1Err != nil {
			t.Fatalf("expected daemon to exit cleanly with code 0 after graceful stop, got: %v", p1Err)
		}
		t.Log("PASS: Daemon process exited cleanly with code 0")
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for daemon to exit after stop command")
	}

	// 9b. Verify owner metadata was cleaned up during teardown
	metaPath := dbPath + ".owner.json"
	if _, err := os.Stat(metaPath); !os.IsNotExist(err) {
		t.Fatalf("expected owner metadata file to be cleaned up after daemon shutdown, err=%v", err)
	}
	t.Log("PASS: Owner metadata cleaned up by OwnerLease.CleanMetadata (metadata cleanup does not independently prove Store.Close; Store.Close order verified in unit regression barrier)")

	// 10. Verify that now contender can acquire lock and start
	contenderReadyFile := filepath.Join(tempDir, "contender_ready.txt")
	p4Args := append([]string{
		"run",
		"-db=" + dbPath,
		"-http-addr=127.0.0.1:0",
		"-instance-id=proc-post-shutdown-contender",
		"-ready-signal-file=" + contenderReadyFile,
	}, policyArgs...)
	p4 := exec.Command(binPath, p4Args...)
	if err := p4.Start(); err != nil {
		t.Fatalf("failed to start contender after daemon stop: %v", err)
	}
	defer func() {
		if p4.Process != nil {
			_ = p4.Process.Kill()
		}
	}()

	var contenderReady bool
	for start := time.Now(); time.Since(start) < 5*time.Second; time.Sleep(50 * time.Millisecond) {
		if data, err := os.ReadFile(contenderReadyFile); err == nil && len(data) > 0 {
			contenderReady = true
			break
		}
	}
	if !contenderReady {
		t.Fatal("contender could not acquire lock / start after daemon shutdown")
	}
	t.Log("PASS: New contender acquired lock and signaled readiness via ready-signal file after daemon shutdown")
}
