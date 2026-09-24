package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/ao"
	"github.com/trungqwe/ai-supervisor/internal/host"
	"github.com/trungqwe/ai-supervisor/internal/recovery"
	"github.com/trungqwe/ai-supervisor/internal/store"
)

type noopObserver struct{}

func (noopObserver) GetWorkerStatus(ctx context.Context, sessionID string) (*ao.WorkerStatus, error) {
	return nil, fmt.Errorf("ao: session %q not found in local bootstrap", sessionID)
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
	operatorPrincipal := fs.String("operator-principal", "", "Authenticated operator principal token")
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

	// Step 1: Prepare DB handle and derive canonical path
	absDB, err := filepath.Abs(filepath.Clean(*dbPath))
	if err != nil {
		return fmt.Errorf("invalid DB path: %w", err)
	}

	var pinned *host.PinnedDB
	if _, err := os.Stat(absDB); errors.Is(err, os.ErrNotExist) {
		// New DB path
		parent := filepath.Dir(absDB)
		name := filepath.Base(absDB)
		pinned, err = host.PrepareNewDB(parent, name)
		if err != nil {
			return fmt.Errorf("failed to prepare new DB: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to stat DB path: %w", err)
	} else {
		// Existing DB path
		pinned, err = host.PrepareExistingDB(absDB)
		if err != nil {
			return fmt.Errorf("failed to prepare existing DB: %w", err)
		}
	}
	defer pinned.Close()

	// Step 2: Acquire ProcessOwnerLease (.owner.lock)
	lease, err := host.AcquireProcessOwnerLease(pinned.CanonicalDBPath, *instanceID)
	if err != nil {
		return fmt.Errorf("failed to acquire owner lease: %w", err)
	}
	defer lease.CloseLockHandle()

	// Step 3: Open Store using validated StoreDBPath (local DOS path)
	ctx := context.Background()
	st, err := store.Open(ctx, store.Config{
		DBPath:        pinned.StoreDBPath,
		BusyTimeoutMs: 5000,
	})
	if err != nil {
		lease.CleanMetadata()
		return fmt.Errorf("failed to open store: %w", err)
	}
	defer st.Close()

	// Step 4: Verify post-open physical identity
	if err := pinned.VerifyPostOpenIdentity(); err != nil {
		lease.CleanMetadata()
		return fmt.Errorf("post-open physical identity verification failed: %w", err)
	}

	// Step 5: Initialize trusted authority
	auth := host.NewAuthority(*operatorPrincipal)

	// Step 6: Setup observer
	var observer recovery.Observer = noopObserver{}
	if *aoAddr != "" {
		aoClient, err := ao.NewClient(*aoAddr, &http.Client{Timeout: policies.SupervisorHTTPTimeout})
		if err != nil {
			lease.CleanMetadata()
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
	_, err = scanner.Run(startupCtx)
	if err != nil {
		lease.CleanMetadata()
		return fmt.Errorf("startup recovery scan failed: %w", err)
	}

	// Step 8: Start readonly HTTP probe server (ZERO effectful routes)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":      "HEALTHY",
			"instance_id": *instanceID,
		})
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if !auth.Available() {
			w.WriteHeader(http.StatusServiceUnavailable)
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
		lease.CleanMetadata()
		return fmt.Errorf("failed to listen on HTTP addr %q: %w", *httpAddr, err)
	}
	defer listener.Close()

	httpServer := &http.Server{Handler: mux}
	go func() {
		_ = httpServer.Serve(listener)
	}()

	// Signal readiness file if requested
	if *readySignalFile != "" {
		_ = os.WriteFile(*readySignalFile, []byte(listener.Addr().String()), 0600)
	}

	// Step 9: Start Named Pipe Server for takeover and status
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
		lease.CleanMetadata()
		return fmt.Errorf("failed to start named pipe server: %w", err)
	}
	defer pipeServer.Close()

	// Wait for OS interrupt or Named Pipe stop signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigCh:
	case <-stopTriggered:
	}

	// Step 10: Execute strict shutdown drain sequence
	drainComponents := host.ShutdownComponents{
		PipeServer: pipeServer,
		HTTPServer: httpServer,
		Authority:  auth,
		Store:      st,
		PinnedDB:   pinned,
		OwnerLease: lease,
	}

	drainErr := host.ExecuteShutdownDrain(10*time.Second, drainComponents)
	if *readySignalFile != "" {
		_ = os.Remove(*readySignalFile)
	}

	return drainErr
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

	// Read sidecar metadata
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

	fmt.Printf("Daemon stopped successfully: status=%s, instance=%s\n", resp.Status, resp.InstanceID)
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
		return fmt.Errorf("daemon status query failed: %w", err)
	}

	fmt.Printf("Daemon status: %s (PID=%d, instance=%s)\n", resp.Status, resp.PID, resp.InstanceID)
	return nil
}
