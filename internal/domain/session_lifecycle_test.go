package domain_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

func TestADR016LifecycleTokensAndNullableModels(t *testing.T) {
	if domain.QuarantineClean != "CLEAN" || domain.QuarantineQuarantined != "QUARANTINED" {
		t.Fatal("quarantine token drift")
	}
	provisioning := []domain.PairProvisioningStage{
		domain.ProvisionRequested, domain.ProvisionConfirmed, domain.ProvisionFailed, domain.ProvisionResolved,
	}
	if got, want := len(provisioning), 4; got != want {
		t.Fatalf("provisioning stage count = %d, want %d", got, want)
	}
	dispatch := []domain.DispatchStage{domain.DispatchBound, domain.SendRequested, domain.SendConfirmed}
	if got, want := len(dispatch), 3; got != want {
		t.Fatalf("dispatch stage count = %d, want %d", got, want)
	}
	stop := []domain.StopStage{
		domain.StopRequested, domain.StopCallSucceeded, domain.StopCallFailed,
		domain.StopTerminationConfirmed, domain.StopTargetAbsent,
	}
	if got, want := len(stop), 5; got != want {
		t.Fatalf("stop stage count = %d, want %d", got, want)
	}

	now := time.Now().UTC()
	operation := domain.StopOperation{
		OperationID:        "stop-1",
		Purpose:            domain.PairMaintenance,
		PairID:             "pair-1",
		SessionID:          "session-1",
		TerminalGeneration: "generation-1",
		Stage:              domain.StopRequested,
		Actor:              "operator",
		RequestedAt:        now,
		ResolutionState:    domain.StopResolutionInFlight,
	}
	raw, err := json.Marshal(operation)
	if err != nil {
		t.Fatalf("marshal nullable stop operation: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal nullable stop operation: %v", err)
	}
	for _, absent := range []string{"task_id", "contract_id", "attempt_id", "resolved_at"} {
		if _, ok := decoded[absent]; ok {
			t.Fatalf("nullable field %q unexpectedly serialized", absent)
		}
	}
}
