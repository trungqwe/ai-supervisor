//go:build windows

package host

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"golang.org/x/sys/windows"
)

var (
	modadvapi32                    = windows.NewLazySystemDLL("advapi32.dll")
	procImpersonateNamedPipeClient = modadvapi32.NewProc("ImpersonateNamedPipeClient")
	procRevertToSelf               = modadvapi32.NewProc("RevertToSelf")
)

type PipeMessageRequest struct {
	Action           string `json:"action"` // "STATUS" or "TAKEOVER"
	CallerInstanceID string `json:"caller_instance_id,omitempty"`
}

type PipeMessageResponse struct {
	Status     string `json:"status"` // "RUNNING", "DRAINED", "UNAUTHORIZED", "ERROR"
	InstanceID string `json:"instance_id,omitempty"`
	PID        int    `json:"pid,omitempty"`
	Error      string `json:"error,omitempty"`
}

// NamedPipeServer serves status and takeover requests with strict caller SID authentication.
type NamedPipeServer struct {
	PipeName        string
	InstanceID      string
	PID             int
	ServerSID       string
	OnStopRequested func() error
	mu              sync.Mutex
	closed          bool
	stopCh          chan struct{}
}

// GetProcessUserSID retrieves the user SID string of the current running process.
func GetProcessUserSID() (string, error) {
	tok, err := windows.OpenCurrentProcessToken()
	if err != nil {
		return "", fmt.Errorf("OpenCurrentProcessToken: %w", err)
	}
	defer tok.Close()

	u, err := tok.GetTokenUser()
	if err != nil {
		return "", fmt.Errorf("GetTokenUser: %w", err)
	}
	return u.User.Sid.String(), nil
}

// StartNamedPipeServer starts listening on the Windows Named Pipe.
func StartNamedPipeServer(pipeName, instanceID string, onStop func() error) (*NamedPipeServer, error) {
	serverSID, err := GetProcessUserSID()
	if err != nil {
		return nil, fmt.Errorf("host: failed to get server process SID: %w", err)
	}

	pipeNameUTF16, err := windows.UTF16PtrFromString(pipeName)
	if err != nil {
		return nil, err
	}

	// Create initial pipe instance synchronously so callers can connect immediately
	hFirst, err := windows.CreateNamedPipe(
		pipeNameUTF16,
		windows.PIPE_ACCESS_DUPLEX,
		windows.PIPE_TYPE_MESSAGE|windows.PIPE_READMODE_MESSAGE|windows.PIPE_WAIT,
		windows.PIPE_UNLIMITED_INSTANCES,
		4096,
		4096,
		0,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("host: failed to create initial named pipe %q: %w", pipeName, err)
	}

	s := &NamedPipeServer{
		PipeName:        pipeName,
		InstanceID:      instanceID,
		PID:             os.Getpid(),
		ServerSID:       serverSID,
		OnStopRequested: onStop,
		stopCh:          make(chan struct{}),
	}

	go s.serveLoop(hFirst)
	return s, nil
}

func (s *NamedPipeServer) serveLoop(initialHandle windows.Handle) {
	pipeNameUTF16, _ := windows.UTF16PtrFromString(s.PipeName)
	currentH := initialHandle

	for {
		// Wait for client connection on current handle
		err := windows.ConnectNamedPipe(currentH, nil)
		if err != nil && !errors.Is(err, windows.ERROR_PIPE_CONNECTED) {
			_ = windows.CloseHandle(currentH)
		} else {
			s.mu.Lock()
			isClosed := s.closed
			s.mu.Unlock()

			if isClosed {
				_ = windows.DisconnectNamedPipe(currentH)
				_ = windows.CloseHandle(currentH)
				return
			}

			// Handle this connected client in background
			go s.handleClient(currentH)
		}

		select {
		case <-s.stopCh:
			return
		default:
		}

		// Create next pipe instance for subsequent connections
		currentH, err = windows.CreateNamedPipe(
			pipeNameUTF16,
			windows.PIPE_ACCESS_DUPLEX,
			windows.PIPE_TYPE_MESSAGE|windows.PIPE_READMODE_MESSAGE|windows.PIPE_WAIT,
			windows.PIPE_UNLIMITED_INSTANCES,
			4096,
			4096,
			0,
			nil,
		)
		if err != nil {
			select {
			case <-s.stopCh:
				return
			case <-time.After(50 * time.Millisecond):
				continue
			}
		}
	}
}

func (s *NamedPipeServer) handleClient(h windows.Handle) {
	defer func() {
		_ = windows.DisconnectNamedPipe(h)
		_ = windows.CloseHandle(h)
	}()

	// Step 1: Read client request first
	buf := make([]byte, 4096)
	var bytesRead uint32
	err := windows.ReadFile(h, buf, &bytesRead, nil)
	if err != nil || bytesRead == 0 {
		return
	}

	// Step 2: Impersonate caller to inspect client thread token SID
	r0, _, callErr := procImpersonateNamedPipeClient.Call(uintptr(h))
	if r0 == 0 {
		s.writePipeResponse(h, PipeMessageResponse{
			Status: "UNAUTHORIZED",
			Error:  fmt.Sprintf("ImpersonateNamedPipeClient failed: %v", callErr),
		})
		return
	}
	defer procRevertToSelf.Call()

	var threadTok windows.Token
	err = windows.OpenThreadToken(windows.CurrentThread(), windows.TOKEN_QUERY, true, &threadTok)
	if err != nil {
		s.writePipeResponse(h, PipeMessageResponse{
			Status: "UNAUTHORIZED",
			Error:  fmt.Sprintf("OpenThreadToken failed: %v", err),
		})
		return
	}
	defer threadTok.Close()

	clientUser, err := threadTok.GetTokenUser()
	if err != nil {
		s.writePipeResponse(h, PipeMessageResponse{
			Status: "UNAUTHORIZED",
			Error:  fmt.Sprintf("GetTokenUser failed: %v", err),
		})
		return
	}
	callerSID := clientUser.User.Sid.String()

	// Revert impersonation immediately after reading identity
	procRevertToSelf.Call()

	// Step 3: SID Verification - must match server process SID
	if callerSID != s.ServerSID {
		resp := PipeMessageResponse{
			Status: "UNAUTHORIZED",
			Error:  "Caller SID mismatch; access denied fail-closed",
		}
		s.writePipeResponse(h, resp)
		return
	}

	// Step 4: Dispatch client request
	var req PipeMessageRequest
	if err := json.Unmarshal(buf[:bytesRead], &req); err != nil {
		s.writePipeResponse(h, PipeMessageResponse{Status: "ERROR", Error: err.Error()})
		return
	}

	switch req.Action {
	case "STATUS":
		resp := PipeMessageResponse{
			Status:     "RUNNING",
			InstanceID: s.InstanceID,
			PID:        s.PID,
		}
		s.writePipeResponse(h, resp)
	case "TAKEOVER":
		if s.OnStopRequested != nil {
			err := s.OnStopRequested()
			if err != nil {
				s.writePipeResponse(h, PipeMessageResponse{Status: "ERROR", Error: err.Error()})
				return
			}
		}
		resp := PipeMessageResponse{
			Status:     "DRAINED",
			InstanceID: s.InstanceID,
			PID:        s.PID,
		}
		s.writePipeResponse(h, resp)
	default:
		s.writePipeResponse(h, PipeMessageResponse{Status: "ERROR", Error: "unknown action"})
	}
}

func (s *NamedPipeServer) writePipeResponse(h windows.Handle, resp PipeMessageResponse) {
	bytes, err := json.Marshal(resp)
	if err != nil {
		return
	}
	var bytesWritten uint32
	_ = windows.WriteFile(h, bytes, &bytesWritten, nil)
	_ = windows.FlushFileBuffers(h)
}

// Close closes the Named Pipe listener and cancels pending serve routines.
// In shutdown drain, this is closed FIRST before waiting for active operations.
func (s *NamedPipeServer) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	close(s.stopCh)
	s.mu.Unlock()

	// Connect a dummy client to unblock ConnectNamedPipe if it is waiting
	pipeUTF16, _ := windows.UTF16PtrFromString(s.PipeName)
	hDummy, _ := windows.CreateFile(
		pipeUTF16,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		0,
		nil,
		windows.OPEN_EXISTING,
		0,
		0,
	)
	if hDummy != windows.InvalidHandle && hDummy != 0 {
		_ = windows.CloseHandle(hDummy)
	}

	return nil
}

// RequestPipeStatus connects to the Named Pipe and queries current daemon status.
func RequestPipeStatus(pipeName string, timeout time.Duration) (*PipeMessageResponse, error) {
	return sendPipeRequest(pipeName, PipeMessageRequest{Action: "STATUS"}, timeout)
}

// RequestPipeTakeover connects to the Named Pipe and requests graceful shutdown drain.
func RequestPipeTakeover(pipeName, callerInstanceID string, timeout time.Duration) (*PipeMessageResponse, error) {
	return sendPipeRequest(pipeName, PipeMessageRequest{
		Action:           "TAKEOVER",
		CallerInstanceID: callerInstanceID,
	}, timeout)
}

func sendPipeRequest(pipeName string, req PipeMessageRequest, timeout time.Duration) (*PipeMessageResponse, error) {
	pipeUTF16, err := windows.UTF16PtrFromString(pipeName)
	if err != nil {
		return nil, err
	}

	deadline := time.Now().Add(timeout)
	var h windows.Handle
	for {
		// Include SECURITY_SQOS_PRESENT | SECURITY_IMPERSONATION to allow server to authenticate caller token SID
		h, err = windows.CreateFile(
			pipeUTF16,
			windows.GENERIC_READ|windows.GENERIC_WRITE,
			0,
			nil,
			windows.OPEN_EXISTING,
			windows.SECURITY_SQOS_PRESENT|windows.SECURITY_IMPERSONATION,
			0,
		)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("host: timeout connecting to pipe %q: %w", pipeName, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	defer windows.CloseHandle(h)

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	var bytesWritten uint32
	if err := windows.WriteFile(h, reqBytes, &bytesWritten, nil); err != nil {
		return nil, fmt.Errorf("host: write to pipe error: %w", err)
	}

	buf := make([]byte, 4096)
	var bytesRead uint32
	if err := windows.ReadFile(h, buf, &bytesRead, nil); err != nil {
		return nil, fmt.Errorf("host: read from pipe error: %w", err)
	}

	var resp PipeMessageResponse
	if err := json.Unmarshal(buf[:bytesRead], &resp); err != nil {
		return nil, fmt.Errorf("host: unmarshal pipe response error: %w", err)
	}

	if resp.Status == "UNAUTHORIZED" {
		return nil, fmt.Errorf("%w: %s", ErrUnauthorizedCaller, resp.Error)
	}

	return &resp, nil
}
