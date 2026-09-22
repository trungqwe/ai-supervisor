package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/trungqwe/ai-supervisor/internal/audit"
	"github.com/trungqwe/ai-supervisor/internal/domain"
)

const (
	AuditDomainSeparator = "ai-supervisor:audit:v1"
	MaxAuditLimit        = 500
)

func writePrefixedString(w io.Writer, s string) {
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(s)))
	w.Write(lenBuf[:])
	if len(s) > 0 {
		io.WriteString(w, s)
	}
}

// ComputeEventHashRaw computes the deterministic SHA-256 hash using domain separation,
// binary sequence encoding, and length-prefixed UTF-8 field framing.
func ComputeEventHashRaw(
	seq int64,
	eventID, eventType, timestampStr, pairID, taskID, contractID, attemptID, actor, detailsJSON, prevHash string,
) string {
	h := sha256.New()
	writePrefixedString(h, AuditDomainSeparator)

	var seqBuf [8]byte
	binary.BigEndian.PutUint64(seqBuf[:], uint64(seq))
	h.Write(seqBuf[:])

	writePrefixedString(h, eventID)
	writePrefixedString(h, eventType)
	writePrefixedString(h, timestampStr)
	writePrefixedString(h, pairID)
	writePrefixedString(h, taskID)
	writePrefixedString(h, contractID)
	writePrefixedString(h, attemptID)
	writePrefixedString(h, actor)
	writePrefixedString(h, detailsJSON)
	writePrefixedString(h, prevHash)

	return hex.EncodeToString(h.Sum(nil))
}

// ComputeEventHash computes the hash for a domain.AuditEvent given its sequence, canonical detailsJSON, and prevHash.
func ComputeEventHash(seq int64, event domain.AuditEvent, detailsJSON string, prevHash string) string {
	tsStr := event.Timestamp.UTC().Format(time.RFC3339Nano)
	return ComputeEventHashRaw(
		seq,
		event.EventID,
		event.EventType,
		tsStr,
		event.PairID,
		event.TaskID,
		event.ContractID,
		event.AttemptID,
		event.Actor,
		detailsJSON,
		prevHash,
	)
}

func (s *Store) withRetry(ctx context.Context, fn func() error) error {
	const maxRetries = 20
	for attempt := 0; attempt < maxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "database is locked") || strings.Contains(errStr, "busy") {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(5+attempt*5) * time.Millisecond):
				continue
			}
		}
		return err
	}
	return fmt.Errorf("store: operation timed out after busy retries")
}

// AppendAuditEvent sanitizes, validates, hashes, and atomically appends an AuditEvent to SQLite.
// Raw secret data is scrubbed before any SQL parameter passing.
func (s *Store) AppendAuditEvent(ctx context.Context, event domain.AuditEvent) (AuditRecord, error) {
	// 1. Validate required fields
	if event.EventID == "" {
		return AuditRecord{}, fmt.Errorf("store: audit event_id is required")
	}
	if event.EventType == "" {
		return AuditRecord{}, fmt.Errorf("store: audit event_type is required")
	}
	if event.Actor == "" {
		return AuditRecord{}, fmt.Errorf("store: audit actor is required")
	}
	if len(event.EventID) > 256 || len(event.EventType) > 256 || len(event.Actor) > 256 ||
		len(event.TaskID) > 256 || len(event.ContractID) > 256 || len(event.AttemptID) > 256 || len(event.PairID) > 256 {
		return AuditRecord{}, fmt.Errorf("store: audit field exceeds maximum length of 256 characters")
	}

	// 2. Deep-copy and sanitize before transaction
	sanitized, detailsJSON, err := audit.SanitizeAuditEvent(event)
	if err != nil {
		return AuditRecord{}, fmt.Errorf("store: failed to sanitize audit event: %w", err)
	}
	if sanitized.Timestamp.IsZero() {
		sanitized.Timestamp = time.Now().UTC()
	} else {
		sanitized.Timestamp = sanitized.Timestamp.UTC()
	}

	// 3. Pre-transaction lineage requirements
	if sanitized.ContractID != "" && sanitized.TaskID == "" {
		return AuditRecord{}, fmt.Errorf("%w: contract_id requires task_id", ErrInvalidAuditLineage)
	}
	if sanitized.AttemptID != "" && (sanitized.TaskID == "" || sanitized.ContractID == "") {
		return AuditRecord{}, fmt.Errorf("%w: attempt_id requires task_id and contract_id", ErrInvalidAuditLineage)
	}

	var rec AuditRecord

	// 4. Atomic append transaction with bounded retry for SQLite busy/locked
	err = s.withRetry(ctx, func() error {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("store: failed to begin audit append transaction: %w", err)
		}
		defer tx.Rollback()

		// Lineage checks against persisted state
		if sanitized.ContractID != "" {
			var exists int
			err := tx.QueryRowContext(ctx, "SELECT 1 FROM task_contracts WHERE contract_id = ? AND task_id = ?",
				sanitized.ContractID, sanitized.TaskID).Scan(&exists)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("%w: contract %s does not belong to task %s", ErrInvalidAuditLineage, sanitized.ContractID, sanitized.TaskID)
				}
				return err
			}
		}
		if sanitized.AttemptID != "" {
			var exists int
			err := tx.QueryRowContext(ctx, "SELECT 1 FROM task_attempts WHERE attempt_id = ? AND task_id = ? AND contract_id = ?",
				sanitized.AttemptID, sanitized.TaskID, sanitized.ContractID).Scan(&exists)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("%w: attempt %s does not belong to task %s and contract %s", ErrInvalidAuditLineage, sanitized.AttemptID, sanitized.TaskID, sanitized.ContractID)
				}
				return err
			}
		}
		if sanitized.PairID != "" && sanitized.TaskID != "" {
			var exists int
			err := tx.QueryRowContext(ctx, "SELECT 1 FROM tasks WHERE task_id = ? AND pair_id = ?",
				sanitized.TaskID, sanitized.PairID).Scan(&exists)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("%w: task %s does not belong to pair %s", ErrInvalidAuditLineage, sanitized.TaskID, sanitized.PairID)
				}
				return err
			}
		}
		if sanitized.TaskID != "" && sanitized.ContractID == "" && sanitized.AttemptID == "" && sanitized.PairID == "" {
			var exists int
			err := tx.QueryRowContext(ctx, "SELECT 1 FROM tasks WHERE task_id = ?", sanitized.TaskID).Scan(&exists)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("%w: task %s not found", ErrInvalidAuditLineage, sanitized.TaskID)
				}
				return err
			}
		}
		if sanitized.PairID != "" && sanitized.TaskID == "" {
			var exists int
			err := tx.QueryRowContext(ctx, "SELECT 1 FROM pairs WHERE pair_id = ?", sanitized.PairID).Scan(&exists)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("%w: pair %s not found", ErrInvalidAuditLineage, sanitized.PairID)
				}
				return err
			}
		}

		// First-write ownership on singleton audit_chain_state
		if _, err := tx.ExecContext(ctx, "UPDATE audit_chain_state SET id = 1 WHERE id = 1"); err != nil {
			return err
		}

		var lastSeq int64
		var lastHash string
		if err := tx.QueryRowContext(ctx, "SELECT last_sequence, last_hash FROM audit_chain_state WHERE id = 1").Scan(&lastSeq, &lastHash); err != nil {
			return fmt.Errorf("store: failed to query audit_chain_state: %w", err)
		}

		nextSeq := lastSeq + 1
		prevHash := lastHash
		eventHash := ComputeEventHash(nextSeq, sanitized, detailsJSON, prevHash)

		var pairID, taskID, contractID, attemptID *string
		if sanitized.PairID != "" {
			pairID = &sanitized.PairID
		}
		if sanitized.TaskID != "" {
			taskID = &sanitized.TaskID
		}
		if sanitized.ContractID != "" {
			contractID = &sanitized.ContractID
		}
		if sanitized.AttemptID != "" {
			attemptID = &sanitized.AttemptID
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO audit_events (
				sequence, event_id, event_type, timestamp,
				pair_id, task_id, contract_id, attempt_id,
				actor, details_json, prev_hash, event_hash
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, nextSeq, sanitized.EventID, sanitized.EventType, sanitized.Timestamp.UTC().Format(time.RFC3339Nano),
			pairID, taskID, contractID, attemptID,
			sanitized.Actor, detailsJSON, prevHash, eventHash)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed") || strings.Contains(err.Error(), "cannot replace existing audit event") {
				return fmt.Errorf("%w: duplicate audit event_id %s: %v", ErrDuplicateKey, sanitized.EventID, err)
			}
			return fmt.Errorf("store: failed to insert audit_event: %w", err)
		}

		if _, err := tx.ExecContext(ctx, "UPDATE audit_chain_state SET last_sequence = ?, last_hash = ? WHERE id = 1", nextSeq, eventHash); err != nil {
			return fmt.Errorf("store: failed to update audit_chain_state: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("store: failed to commit audit append transaction: %w", err)
		}

		rec = AuditRecord{
			Sequence:  nextSeq,
			Event:     sanitized,
			PrevHash:  prevHash,
			EventHash: eventHash,
		}
		return nil
	})

	if err != nil {
		return AuditRecord{}, err
	}
	return rec, nil
}

// GetAuditEvent retrieves a single audit record by its unique eventID.
func (s *Store) GetAuditEvent(ctx context.Context, eventID string) (AuditRecord, error) {
	if eventID == "" {
		return AuditRecord{}, fmt.Errorf("store: eventID is required")
	}

	var rec AuditRecord
	var tsStr string
	var pairID, taskID, contractID, attemptID sql.NullString
	var detailsJSON string

	row := s.db.QueryRowContext(ctx, `
		SELECT sequence, event_id, event_type, timestamp,
		       pair_id, task_id, contract_id, attempt_id,
		       actor, details_json, prev_hash, event_hash
		FROM audit_events
		WHERE event_id = ?
	`, eventID)

	err := row.Scan(
		&rec.Sequence,
		&rec.Event.EventID,
		&rec.Event.EventType,
		&tsStr,
		&pairID,
		&taskID,
		&contractID,
		&attemptID,
		&rec.Event.Actor,
		&detailsJSON,
		&rec.PrevHash,
		&rec.EventHash,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AuditRecord{}, ErrAuditEventNotFound
		}
		return AuditRecord{}, fmt.Errorf("store: failed to query audit_event: %w", err)
	}

	t, err := time.Parse(time.RFC3339Nano, tsStr)
	if err != nil {
		return AuditRecord{}, fmt.Errorf("store: corrupt timestamp in audit_events: %w", err)
	}
	rec.Event.Timestamp = t.UTC()

	if pairID.Valid {
		rec.Event.PairID = pairID.String
	}
	if taskID.Valid {
		rec.Event.TaskID = taskID.String
	}
	if contractID.Valid {
		rec.Event.ContractID = contractID.String
	}
	if attemptID.Valid {
		rec.Event.AttemptID = attemptID.String
	}

	if detailsJSON != "" && detailsJSON != "{}" {
		dec := json.NewDecoder(strings.NewReader(detailsJSON))
		dec.UseNumber()
		var det map[string]any
		if err := dec.Decode(&det); err != nil {
			return AuditRecord{}, fmt.Errorf("store: corrupt details_json in audit_events: %w", err)
		}
		rec.Event.Details = det
	} else {
		rec.Event.Details = make(map[string]any)
	}

	return rec, nil
}

// ListAuditEvents retrieves up to limit audit records with sequence > afterSequence in ascending sequence order.
func (s *Store) ListAuditEvents(ctx context.Context, afterSequence int64, limit int) ([]AuditRecord, error) {
	if limit <= 0 || limit > MaxAuditLimit {
		return nil, fmt.Errorf("%w: limit must be between 1 and %d, got %d", ErrInvalidLimit, MaxAuditLimit, limit)
	}
	if afterSequence < 0 {
		afterSequence = 0
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT sequence, event_id, event_type, timestamp,
		       pair_id, task_id, contract_id, attempt_id,
		       actor, details_json, prev_hash, event_hash
		FROM audit_events
		WHERE sequence > ?
		ORDER BY sequence ASC
		LIMIT ?
	`, afterSequence, limit)
	if err != nil {
		return nil, fmt.Errorf("store: failed to list audit_events: %w", err)
	}
	defer rows.Close()

	records := make([]AuditRecord, 0, limit)
	for rows.Next() {
		var rec AuditRecord
		var tsStr string
		var pairID, taskID, contractID, attemptID sql.NullString
		var detailsJSON string

		err := rows.Scan(
			&rec.Sequence,
			&rec.Event.EventID,
			&rec.Event.EventType,
			&tsStr,
			&pairID,
			&taskID,
			&contractID,
			&attemptID,
			&rec.Event.Actor,
			&detailsJSON,
			&rec.PrevHash,
			&rec.EventHash,
		)
		if err != nil {
			return nil, fmt.Errorf("store: failed to scan audit_event row: %w", err)
		}

		t, err := time.Parse(time.RFC3339Nano, tsStr)
		if err != nil {
			return nil, fmt.Errorf("store: corrupt timestamp in audit_events: %w", err)
		}
		rec.Event.Timestamp = t.UTC()

		if pairID.Valid {
			rec.Event.PairID = pairID.String
		}
		if taskID.Valid {
			rec.Event.TaskID = taskID.String
		}
		if contractID.Valid {
			rec.Event.ContractID = contractID.String
		}
		if attemptID.Valid {
			rec.Event.AttemptID = attemptID.String
		}

		if detailsJSON != "" && detailsJSON != "{}" {
			dec := json.NewDecoder(strings.NewReader(detailsJSON))
			dec.UseNumber()
			var det map[string]any
			if err := dec.Decode(&det); err != nil {
				return nil, fmt.Errorf("store: corrupt details_json in audit_events: %w", err)
			}
			rec.Event.Details = det
		} else {
			rec.Event.Details = make(map[string]any)
		}

		records = append(records, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: rows iteration error: %w", err)
	}

	return records, nil
}

// VerifyAuditChain validates the tamper evidence of the append-only audit chain.
// It verifies contiguous sequence numbers from 1, genesis hash, prev_hash linking,
// exact event_hash recomputations, and alignment with persisted audit_chain_state head.
func (s *Store) VerifyAuditChain(ctx context.Context) error {
	var headSeq int64
	var headHash string
	if err := s.db.QueryRowContext(ctx, "SELECT last_sequence, last_hash FROM audit_chain_state WHERE id = 1").Scan(&headSeq, &headHash); err != nil {
		return fmt.Errorf("%w: failed to read audit_chain_state: %v", ErrAuditChainInvalid, err)
	}

	if headSeq == 0 {
		if headHash != GenesisAuditHash {
			return fmt.Errorf("%w: empty chain has invalid head_hash %q (expected %q)", ErrAuditChainInvalid, headHash, GenesisAuditHash)
		}
		var count int
		if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM audit_events").Scan(&count); err != nil {
			return fmt.Errorf("%w: failed to count audit_events: %v", ErrAuditChainInvalid, err)
		}
		if count != 0 {
			return fmt.Errorf("%w: head_sequence is 0 but %d audit events exist", ErrAuditChainInvalid, count)
		}
		return nil
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT sequence, event_id, event_type, timestamp,
		       COALESCE(pair_id, ''), COALESCE(task_id, ''), COALESCE(contract_id, ''), COALESCE(attempt_id, ''),
		       actor, details_json, prev_hash, event_hash
		FROM audit_events
		ORDER BY sequence ASC
	`)
	if err != nil {
		return fmt.Errorf("%w: failed to query audit_events: %v", ErrAuditChainInvalid, err)
	}
	defer rows.Close()

	expectedSeq := int64(1)
	expectedPrevHash := GenesisAuditHash

	for rows.Next() {
		var seq int64
		var eventID, eventType, tsStr, pairID, taskID, contractID, attemptID, actor, detailsJSON, prevHash, eventHash string

		err := rows.Scan(
			&seq,
			&eventID,
			&eventType,
			&tsStr,
			&pairID,
			&taskID,
			&contractID,
			&attemptID,
			&actor,
			&detailsJSON,
			&prevHash,
			&eventHash,
		)
		if err != nil {
			return fmt.Errorf("%w: failed to scan audit event: %v", ErrAuditChainInvalid, err)
		}

		if seq != expectedSeq {
			return fmt.Errorf("%w: sequence discontinuity: expected %d, got %d", ErrAuditChainInvalid, expectedSeq, seq)
		}

		if prevHash != expectedPrevHash {
			return fmt.Errorf("%w: prev_hash mismatch at sequence %d: expected %q, got %q", ErrAuditChainInvalid, seq, expectedPrevHash, prevHash)
		}

		recomputed := ComputeEventHashRaw(
			seq, eventID, eventType, tsStr, pairID, taskID, contractID, attemptID, actor, detailsJSON, prevHash,
		)
		if eventHash != recomputed {
			return fmt.Errorf("%w: event_hash mismatch at sequence %d: stored %q, recomputed %q", ErrAuditChainInvalid, seq, eventHash, recomputed)
		}

		expectedPrevHash = eventHash
		expectedSeq++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("%w: rows iteration error: %v", ErrAuditChainInvalid, err)
	}

	actualCount := expectedSeq - 1
	if actualCount != headSeq {
		return fmt.Errorf("%w: chain state sequence %d does not match event count %d", ErrAuditChainInvalid, headSeq, actualCount)
	}

	if expectedPrevHash != headHash {
		return fmt.Errorf("%w: chain state last_hash %q does not match final event hash %q", ErrAuditChainInvalid, headHash, expectedPrevHash)
	}

	return nil
}
