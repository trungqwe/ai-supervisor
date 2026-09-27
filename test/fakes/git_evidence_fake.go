package fakes

import (
	"context"

	"github.com/trungqwe/ai-supervisor/internal/domain"
)

// FakeGitEvidenceCollector implements domain.GitEvidenceCollector for tests.
type FakeGitEvidenceCollector struct {
	Result domain.GitEvidenceResult
	Err    error
}

func NewFakeGitEvidenceCollector(result domain.GitEvidenceResult) *FakeGitEvidenceCollector {
	return &FakeGitEvidenceCollector{
		Result: result,
	}
}

func (f *FakeGitEvidenceCollector) Collect(ctx context.Context, worktreePath string) (domain.GitEvidenceResult, error) {
	if f.Err != nil {
		return domain.GitEvidenceResult{}, f.Err
	}
	return f.Result, nil
}
