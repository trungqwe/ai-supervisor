package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/ao"
	"github.com/trungqwe/ai-supervisor/internal/host"
	"github.com/trungqwe/ai-supervisor/internal/recovery"
	"github.com/trungqwe/ai-supervisor/internal/stop"
	"github.com/trungqwe/ai-supervisor/internal/store"
)

type daemonAO interface {
	GetWorkerStatus(context.Context, string) (*ao.WorkerStatus, error)
	StopWorker(context.Context, string) (*ao.StopWorkerResult, error)
}

type noopObserver struct{}

func (noopObserver) GetWorkerStatus(ctx context.Context, sessionID string) (*ao.WorkerStatus, error) {
	return nil, fmt.Errorf("ao: session %q not found in local bootstrap", sessionID)
}

func (noopObserver) StopWorker(ctx context.Context, sessionID string) (*ao.StopWorkerResult, error) {
	return nil, fmt.Errorf("ao: session %q cannot be stopped in local bootstrap", sessionID)
}

type fnCloser struct {
	fn func() error
}

func (f *fnCloser) Close() error {
	if f == nil || f.fn == nil {
		return nil
	}
	return f.fn()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: supervisor <run|stop|status> [flags]\n")
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "run":
		if err := runDaemon(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "supervisor run failed: %v\n", err)
			if errors.Is(err, host.ErrSharingViolation) {
				os.Exit(32) // ERROR_SHARING_VIOLATION exit code
			}
			os.Exit(2)
		}
	case "stop":
		if err := stopDaemon(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "supervisor stop failed: %v\n", err)
			os.Exit(1)
		}
	case "status":
		if err := statusDaemon(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "supervisor status failed: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s. Supported: run, stop, status\n", command)
		os.Exit(1)
	}
}

func runDaemon(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)

	dbPath := fs.String("db", "", "Path to SQLite database file (required)")
	httpAddr := fs.String("http-addr", "127.0.0.1:0", "Address for readonly HTTP probe server")
	aoAddr := fs.String("ao-addr", "", "Address of Untrivial AO REST daemon (optional)")
	instanceID := fs.String("instance-id", "", "Unique daemon instance ID (defaults to timestamp-pid)")
	_ = fs.String("operator-principal", "", "Operator principal token (unverified in V1, remains OPEN dependency per R1-003)")
	readySignalFile := fs.String("ready-signal-file", "", "Optional file written after startup readiness achieved")

	// 8 mandatory operational policies
	httpTimeout := fs.Duration("http-timeout", 0, "SUPERVISOR_HTTP_TIMEOUT (must be positive)")
	healthTimeout := fs.Duration("health-timeout", 0, "SUPERVISOR_HEALTH_PROBE_TIMEOUT (must be positive)")
	spawnTimeout := fs.Duration("spawn-timeout", 0, "SUPERVISOR_SPAWN_TIMEOUT (must be positive)")
	sendTimeout := fs.Duration("send-timeout", 0, "SUPERVISOR_SEND_TIMEOUT (must be positive)")
	pollInterval := fs.Duration("poll-interval", 0, "SUPERVISOR_ACTIVITY_POLL_INTERVAL (must be positive)")
	deadline := fs.Duration("deadline", 0, "SUPERVISOR_EXECUTION_DEADLINE (must be positive)")
	killTimeout := fs.Duration("kill-timeout", 0, "SUPERVISOR_KILL_STOP_TIMEOUT (must be positive)")
	workspaceTimeout := fs.Duration("workspace-timeout", 0, "SUPERVISOR_WORKSPACE_READ_TIMEOUT (must be positive)")

	// Test/diagnostic hooks for deterministic fault injection and barrier testing
	testHoldPermitDuration := fs.Duration("test-hold-permit-duration", 0, "Test hook: hold an active permit to test drain timeout")
	testPauseBeforeCreateDuration := fs.Duration("test-pause-before-create-duration", 0, "Test hook: pause after lock acquisition before CREATE_NEW")
	testLockAcquiredSignal := fs.String("test-lock-acquired-signal", "", "Test hook: write file when owner lock is acquired before CREATE_NEW")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *dbPath == "" {
		return errors.New("missing mandatory -db flag")
	}

	policies := host.Policies{
		SupervisorHTTPTimeout:          *httpTimeout,
		SupervisorHealthProbeTimeout:   *healthTimeout,
		SupervisorSpawnTimeout:         *spawnTimeout,
		SupervisorSendTimeout:          *sendTimeout,
		SupervisorActivityPollInterval: *pollInterval,
		SupervisorExecutionDeadline:    *deadline,
		SupervisorKillStopTimeout:      *killTimeout,
		SupervisorWorkspaceReadTimeout: *workspaceTimeout,
	}

	// Validate 8 policies fail closed if any are unset
	if err := policies.Validate(); err != nil {
		return fmt.Errorf("policy validation failed: %w", err)
	}

	if *instanceID == "" {
		*instanceID = fmt.Sprintf("daemon-%d-%d", time.Now().UnixNano(), os.Getpid())
	}

	absDB, err := filepath.Abs(filepath.Clean(*dbPath))
	if err != nil {
		return fmt.Errorf("invalid DB path: %w", err)
	}

	var (
		cleanupOnEarlyErr = true
		pinned            *host.PinnedDB
		lease             *host.ProcessOwnerLease
		st                *store.Store
	)

	// Early error cleanup defer: only active before main loop / ExecuteShutdownDrain takes over (R1-002)
	defer func() {
		if cleanupOnEarlyErr {
			if st != nil {
				_ = st.Close()
			}
			if pinned != nil {
				_ = pinned.Close()
			}
			if lease != nil {
				lease.CleanMetadata()
				_ = lease.CloseLockHandle()
			}
		}
	}()

	// Step 1 & 2: Handle DB preparation and acquire ProcessOwnerLease (.owner.lock)
	// Invariant (R1-001): For a new DB, acquire owner lease BEFORE CREATE_NEW.
	if _, err := os.Stat(absDB); errors.Is(err, os.ErrNotExist) {
		// New DB path
		parent := filepath.Dir(absDB)
		name := filepath.Base(absDB)
		newPrep, err := host.ResolveNewDBPaths(parent, name)
		if err != nil {
			return fmt.Errorf("failed to resolve new DB paths: %w", err)
		}

		// Acquire owner lock FIRST before creating the file
		lease, err = host.AcquireProcessOwnerLease(newPrep.CanonicalDBPath, *instanceID)
		if err != nil {
			return fmt.Errorf("failed to acquire owner lease for new DB: %w", err)
		}

		// Barrier probe (R1-001): signal that lock is held while DB file does NOT exist yet
		if *testLockAcquiredSignal != "" {
			_ = os.WriteFile(*testLockAcquiredSignal, []byte("LOCKED"), 0600)
		}
		if *testPauseBeforeCreateDuration > 0 {
			time.Sleep(*testPauseBeforeCreateDuration)
		}

		// Now create 0-byte file exclusively via CREATE_NEW and pin without FILE_SHARE_DELETE
		pinned, err = newPrep.CreateAndPinDB()
		if err != nil {
			return fmt.Errorf("failed to create and pin new DB: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to stat DB path: %w", err)
	} else {
		// Existing DB path: open and pin handle first
		pinned, err = host.PrepareExistingDB(absDB)
		if err != nil {
			return fmt.Errorf("failed to prepare existing DB: %w", err)
		}

		lease, err = host.AcquireProcessOwnerLease(pinned.CanonicalDBPath, *instanceID)
		if err != nil {
			return fmt.Errorf("failed to acquire owner lease for existing DB: %w", err)
		}
	}

	// Step 3: Open Store using validated StoreDBPath (local DOS path)
	ctx := context.Background()
	st, err = store.Open(ctx, store.Config{
		DBPath:        pinned.StoreDBPath,
		BusyTimeoutMs: 5000,
	})
	if err != nil {
		return fmt.Errorf("failed to open store: %w", err)
	}

	// Step 4: Verify post-open physical identity
	if err := pinned.VerifyPostOpenIdentity(); err != nil {
		return fmt.Errorf("post-open physical identity verification failed: %w", err)
	}

	// Step 5: Initialize trusted authority
	// Invariant (R1-003): Verified operator principal remains an OPEN dependency at trusted boundary.
	auth := host.NewAuthority("")

	// Step 6: Setup observer
	var observer daemonAO = noopObserver{}
	if *aoAddr != "" {
		aoClient, err := ao.NewClient(*aoAddr, &http.Client{Timeout: policies.SupervisorHTTPTimeout})
		if err != nil {
			return fmt.Errorf("failed to initialize AO client: %w", err)
		}
		observer = aoClient
	}

	// Step 7: Startup-before-serve: run recovery scanner before opening any listener
	scanner := &recovery.Runner{
		Store:                st,
		AO:                   observer,
		Host:                 auth,
		ActivityPollInterval: policies.SupervisorActivityPollInterval,
		ExecutionDeadline:    policies.SupervisorExecutionDeadline,
		Actor:                "HOST_BOOTSTRAP",
		Now:                  time.Now,
	}

	startupCtx, cancelStartup := context.WithTimeout(ctx, 30*time.Second)
	defer cancelStartup()
	scanReport, err := scanner.Run(startupCtx)
	if err != nil {
		return fmt.Errorf("startup recovery scan failed: %w", err)
	}
	// Invariant (R1-004): Scanner must report Complete == true before service begins
	if !scanReport.Complete {
		return fmt.Errorf("startup recovery scan incomplete (PendingAO=%v, Classified=%d)", scanReport.PendingAO, scanReport.Classified)
	}

	// Step 8: Start background poller and TimeoutMonitor scheduler (R1-004)
	poller := &recovery.Poller{
		Store:    st,
		AO:       observer,
		Owner:    scanner,
		Interval: policies.SupervisorActivityPollInterval,
		Actor:    "HOST_POLLER",
	}
	pollerCtx, cancelPoller := context.WithCancel(ctx)
	if err := poller.Start(pollerCtx); err != nil {
		return fmt.Errorf("failed to start poller: %w", err)
	}

	stopCoord := &stop.Coordinator{
		Store:       st,
		AO:          observer,
		Operator:    auth,
		KillTimeout: policies.SupervisorKillStopTimeout,
		TimeoutHost: auth,
		Now:         time.Now,
	}
	timeoutMonitor := &recovery.TimeoutMonitor{
		Owner:    scanner,
		Stop:     stopCoord,
		Interval: policies.SupervisorActivityPollInterval,
		Actor:    "TIMEOUT_MONITOR",
		Now:      time.Now,
	}

	timeoutCtx, cancelTimeout := context.WithCancel(ctx)
	timeoutDone := make(chan struct{})

	// Health monitoring for background schedulers (R1-004)
	var (
		healthMu      sync.RWMutex
		daemonHealthy = true
	)
	setUnhealthy := func(component string, failure error) {
		healthMu.Lock()
		daemonHealthy = false
		healthMu.Unlock()
		auth.SetUnavailable()
		fmt.Fprintf(os.Stderr, "daemon component %s failed fail-closed: %v\n", component, failure)
	}

	go func() {
		defer close(timeoutDone)
		ticker := time.NewTicker(policies.SupervisorActivityPollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-timeoutCtx.Done():
				return
			case <-ticker.C:
				if err := timeoutMonitor.Tick(timeoutCtx); err != nil && timeoutCtx.Err() == nil {
					setUnhealthy("TimeoutMonitor", err)
				}
			}
		}
	}()

	// Note on Poller failure notification (R1-004 / BLOCKER-P03-004-POLLER-ASYNC-NOTIFICATION):
	// recovery.Poller runs in a background goroutine launched by poller.Start(ctx).
	// Because recovery.Poller's done channel and lastErr are unexported and recovery is in
	// forbidden_scope, the host does not have an asynchronous notification channel while
	// running. The host joins poller during shutdown drain via poller.Stop().

	pollerCloser := &fnCloser{fn: func() error {
		cancelPoller()
		return poller.Stop()
	}}
	timeoutCloser := &fnCloser{fn: func() error {
		cancelTimeout()
		<-timeoutDone
		return nil
	}}

	// Step 9: Start Named Pipe Server for takeover and status
	// Invariant (R1-004): Must be initialized and ready BEFORE writing ready signal or serving probe
	stopTriggered := make(chan struct{})
	var stopOnce bool
	pipeServer, err := host.StartNamedPipeServer(lease.PipeName, *instanceID, func() error {
		if !stopOnce {
			stopOnce = true
			close(stopTriggered)
		}
		return nil
	})
	if err != nil {
		_ = pollerCloser.Close()
		_ = timeoutCloser.Close()
		return fmt.Errorf("failed to start named pipe server: %w", err)
	}

	// Step 10: Start readonly HTTP probe server (ZERO effectful routes)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":      "HEALTHY",
			"instance_id": *instanceID,
		})
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		healthMu.RLock()
		healthy := daemonHealthy
		healthMu.RUnlock()

		// Invariant (R1-004): Readiness requires authority available, scan complete, and healthy background schedulers
		if !healthy || !auth.Available() || !scanReport.Complete {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "UNAVAILABLE",
				"error":  "host admission closed or background component failed",
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":      "READY",
			"instance_id": *instanceID,
		})
	})

	listener, err := net.Listen("tcp", *httpAddr)
	if err != nil {
		_ = pipeServer.Close()
		_ = pollerCloser.Close()
		_ = timeoutCloser.Close()
		return fmt.Errorf("failed to listen on HTTP addr %q: %w", *httpAddr, err)
	}

	httpServer := &http.Server{Handler: mux}
	go func() {
		_ = httpServer.Serve(listener)
	}()

	// Invariant (R1-004): Only write readySignalFile AFTER scanner Complete, poller, scheduler, pipe, and HTTP all ready
	if *readySignalFile != "" {
		_ = os.WriteFile(*readySignalFile, []byte(listener.Addr().String()), 0600)
	}

	// Test hook: hold active permit to test real binary drain timeout (R1-002)
	if *testHoldPermitDuration > 0 {
		permit, err := auth.AcquireExclusiveScope(ctx, "test-hold-pair", "IN_FLIGHT_WORKER")
		if err != nil {
			return fmt.Errorf("failed to acquire test hold permit: %w", err)
		}
		go func() {
			time.Sleep(*testHoldPermitDuration)
			_ = permit.Release()
		}()
	}

	// Wait for OS interrupt or Named Pipe stop signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigCh:
	case <-stopTriggered:
	}

	// Step 11: Execute strict shutdown drain sequence (R1-002, R1-004)
	// Disarm early error defer so ExecuteShutdownDrain owns teardown
	cleanupOnEarlyErr = false

	drainComponents := host.ShutdownComponents{
		PipeServer: pipeServer,
		HTTPServer: httpServer,
		Authority:  auth,
		Schedulers: []io.Closer{pollerCloser, timeoutCloser}, // stopped before Store.Close
		Store:      st,
		PinnedDB:   pinned,
		OwnerLease: lease,
	}

	drainErr := host.ExecuteShutdownDrain(10*time.Second, drainComponents)
	if *readySignalFile != "" {
		_ = os.Remove(*readySignalFile)
	}

	if drainErr != nil {
		// R1-002: When drain timed out, ExecuteShutdownDrain blocked until all active
		// permit holders joined, then completed normal teardown (Store, PinnedDB, lock).
		// Resources have been properly released. Return the drain error for logging.
		return fmt.Errorf("shutdown drain completed with timeout: %w", drainErr)
	}

	return nil
}

func stopDaemon(args []string) error {
	fs := flag.NewFlagSet("stop", flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to SQLite database file")
	timeout := fs.Duration("timeout", 10*time.Second, "Timeout waiting for stop drain")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dbPath == "" {
		return errors.New("missing mandatory -db flag")
	}

	absDB, err := filepath.Abs(filepath.Clean(*dbPath))
	if err != nil {
		return err
	}

	_, metaPath, pipeName := host.DeriveSidecarPaths(absDB)
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return fmt.Errorf("failed to read owner metadata %q: %w", metaPath, err)
	}
	var meta host.OwnerMetadata
	if err := json.Unmarshal(data, &meta); err == nil && meta.PipeName != "" {
		pipeName = meta.PipeName
	}

	resp, err := host.RequestPipeTakeover(pipeName, "cli-stop", *timeout)
	if err != nil {
		return fmt.Errorf("pipe takeover failed: %w", err)
	}
	if resp == nil || resp.Status != "STOP_ACKNOWLEDGED" {
		return fmt.Errorf("pipe takeover rejected: status=%v", resp)
	}

	fmt.Printf("Stop request acknowledged: status=%s, instance=%s\n", resp.Status, resp.InstanceID)
	return nil
}

func statusDaemon(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	dbPath := fs.String("db", "", "Path to SQLite database file")
	timeout := fs.Duration("timeout", 5*time.Second, "Timeout waiting for status")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dbPath == "" {
		return errors.New("missing mandatory -db flag")
	}

	absDB, err := filepath.Abs(filepath.Clean(*dbPath))
	if err != nil {
		return err
	}

	_, metaPath, pipeName := host.DeriveSidecarPaths(absDB)
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return fmt.Errorf("daemon is not running (no metadata found at %q)", metaPath)
	}

	var meta host.OwnerMetadata
	if err := json.Unmarshal(data, &meta); err == nil && meta.PipeName != "" {
		pipeName = meta.PipeName
	}

	resp, err := host.RequestPipeStatus(pipeName, *timeout)
	if err != nil {
		return fmt.Errorf("pipe status query failed: %w", err)
	}

	fmt.Printf("Daemon status: %s (instance=%s, pid=%d)\n", resp.Status, resp.InstanceID, resp.PID)
	return nil
}
