package store

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/trungqwe/ai-supervisor/internal/domain"
	"modernc.org/sqlite"
)

// InsertTaskContract persists a new TaskContract and enforces durable revision lineage invariants (ADR-012)
// and task lifecycle state authority (Finding R3-002).
func (s *Store) InsertTaskContract(ctx context.Context, c domain.TaskContract) error {
	if strings.TrimSpace(c.ContractID) == "" {
		return errors.New("store: contract_id must not be empty")
	}
	if strings.TrimSpace(c.TaskID) == "" {
		return errors.New("store: task_id must not be empty")
	}
	if c.RevisionNumber < 1 {
		return fmt.Errorf("store: revision_number must be >= 1 (got %d)", c.RevisionNumber)
	}
	if strings.TrimSpace(c.BaseSHA) == "" {
		return errors.New("store: base_sha must not be empty")
	}

	// Finding R2-005: New contracts must start mutable; contract freezing belongs to PrepareDispatch
	if c.IsImmutable {
		return fmt.Errorf("%w: new task contract must not be immutable on insert", ErrInvalidContractLineage)
	}

	// Prepare JSON serialization before opening transaction
	payloadBytes, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("store: failed to serialize task contract: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: failed to begin contract insert transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Read task state
	var taskStateStr string
	err = tx.QueryRowContext(ctx, "SELECT state FROM tasks WHERE task_id = ?", c.TaskID).Scan(&taskStateStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTaskNotFound
		}
		return fmt.Errorf("store: failed to query task state for contract insert: %w", err)
	}
	taskState := domain.TaskState(taskStateStr)

	// 2. Validate allowed insertion state (Finding R3-002)
	// Revision 1 requires task state = DRAFT
	// Revision > 1 requires task state = REVISION_REQUIRED or DRAFT (supporting human replanning)
	// All other states reject insertion with ErrContractInsertState
	if c.RevisionNumber == 1 {
		if taskState != domain.StateDraft {
			return fmt.Errorf("%w: contract revision 1 requires task state DRAFT, got %q", ErrContractInsertState, taskState)
		}
		if c.SupersedesContractID != nil && strings.TrimSpace(*c.SupersedesContractID) != "" {
			return fmt.Errorf("%w: revision 1 must not specify supersedes_contract_id", ErrInvalidContractLineage)
		}
	} else {
		if taskState != domain.StateRevisionRequired && taskState != domain.StateDraft {
			return fmt.Errorf("%w: contract revision %d requires task state REVISION_REQUIRED or DRAFT, got %q", ErrContractInsertState, c.RevisionNumber, taskState)
		}

		if c.SupersedesContractID == nil || strings.TrimSpace(*c.SupersedesContractID) == "" {
			return fmt.Errorf("%w: revision %d must specify supersedes_contract_id", ErrInvalidContractLineage, c.RevisionNumber)
		}

		prevID := strings.TrimSpace(*c.SupersedesContractID)
		var prevTaskID string
		var prevRev int
		var prevBaseSHA string
		var prevIsImmutable int

		queryPrev := `
SELECT task_id, revision_number, base_sha, is_immutable
FROM task_contracts
WHERE contract_id = ?
`
		err := tx.QueryRowContext(ctx, queryPrev, prevID).Scan(&prevTaskID, &prevRev, &prevBaseSHA, &prevIsImmutable)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("%w: superseded contract %q not found", ErrInvalidContractLineage, prevID)
			}
			return fmt.Errorf("store: failed to query superseded contract: %w", err)
		}

		if prevTaskID != c.TaskID {
			return fmt.Errorf("%w: superseded contract belongs to task %q, not %q", ErrInvalidContractLineage, prevTaskID, c.TaskID)
		}
		if prevRev != c.RevisionNumber-1 {
			return fmt.Errorf("%w: revision jump: previous revision is %d, expected %d", ErrInvalidContractLineage, prevRev, c.RevisionNumber-1)
		}
		if prevBaseSHA != c.BaseSHA {
			return fmt.Errorf("%w: base_sha changed from %q to %q across revisions", ErrInvalidContractLineage, prevBaseSHA, c.BaseSHA)
		}
		if prevIsImmutable != 1 {
			return fmt.Errorf("%w: previous contract revision %d is not frozen/immutable", ErrInvalidContractLineage, prevRev)
		}
	}

	query := `
INSERT INTO task_contracts (
    contract_id, task_id, revision_number, supersedes_contract_id,
    base_sha, payload_json, is_immutable
) VALUES (?, ?, ?, ?, ?, ?, 0)
`

	_, err = tx.ExecContext(ctx, query,
		c.ContractID,
		c.TaskID,
		c.RevisionNumber,
		c.SupersedesContractID,
		c.BaseSHA,
		string(payloadBytes),
	)
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) {
			if strings.Contains(sqliteErr.Error(), "FOREIGN KEY constraint failed") {
				return fmt.Errorf("%w: foreign key violation inserting contract", ErrForeignKeyViolation)
			}
			if strings.Contains(sqliteErr.Error(), "UNIQUE constraint failed") {
				return fmt.Errorf("%w: duplicate contract or revision conflict", ErrDuplicateKey)
			}
		}
		return fmt.Errorf("store: failed to insert task contract: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: failed to commit contract insert: %w", err)
	}

	return nil
}

// GetTaskContract retrieves a TaskContract, unmarshaling the JSON payload with UseNumber() to preserve large integer precision.
func (s *Store) GetTaskContract(ctx context.Context, contractID string) (domain.TaskContract, error) {
	query := `
SELECT payload_json, is_immutable
FROM task_contracts
WHERE contract_id = ?
`

	var payloadJSON string
	var isImmutableInt int

	err := s.db.QueryRowContext(ctx, query, contractID).Scan(&payloadJSON, &isImmutableInt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.TaskContract{}, ErrContractNotFound
		}
		return domain.TaskContract{}, fmt.Errorf("store: failed to query task contract: %w", err)
	}

	var c domain.TaskContract
	dec := json.NewDecoder(bytes.NewReader([]byte(payloadJSON)))
	dec.UseNumber()
	if err := dec.Decode(&c); err != nil {
		return domain.TaskContract{}, fmt.Errorf("store: failed to decode contract payload: %w", err)
	}

	c.IsImmutable = (isImmutableInt == 1)
	return c, nil
}
