package workflowprobe

import (
	"errors"
	"fmt"
	"io"

	"solo-0014-zone-rollout-control/internal/domain"
)

func Run(args []string, output io.Writer) error {
	if len(args) != 1 {
		return errors.New("expected exactly one workflow name")
	}
	var run func() error
	switch args[0] {
	case "plan-revision-lifecycle":
		run = runPlanRevisionLifecycle
	case "node-lease-allocation":
		run = runNodeLeaseAllocation
	case "probe-ingestion":
		run = runProbeIngestion
	case "consistency-assessment":
		run = runConsistencyAssessment
	default:
		return fmt.Errorf("unsupported workflow %q", args[0])
	}
	if err := run(); err != nil {
		return fmt.Errorf("%s: %w", args[0], err)
	}
	_, _ = fmt.Fprintf(output, "%s: passed\n", args[0])
	return nil
}

type apiFailure struct {
	Status  int
	Code    string
	Message string
	Details map[string]any
}

func (e *apiFailure) Error() string {
	return fmt.Sprintf("HTTP %d %s: %s", e.Status, e.Code, e.Message)
}

func expectFailure(err error, status int, code string) error {
	var failure *apiFailure
	if !errors.As(err, &failure) {
		if err == nil {
			return fmt.Errorf("expected HTTP %d %s but request succeeded", status, code)
		}
		return fmt.Errorf("expected HTTP %d %s but got %v", status, code, err)
	}
	if failure.Status != status || failure.Code != code {
		return fmt.Errorf("expected HTTP %d %s, got HTTP %d %s", status, code, failure.Status, failure.Code)
	}
	return nil
}

// expectConflictDetails asserts that an idempotency conflict describes the
// original request with a stable digest, the bound entity, and the names of
// the key fields that differ.
func expectConflictDetails(
	err error,
	entityKind string,
	entityID string,
	digest string,
	recordedAt string,
	differing ...string,
) error {
	var failure *apiFailure
	if !errors.As(err, &failure) {
		return fmt.Errorf("expected idempotency conflict but got %v", err)
	}
	if failure.Code != string(domain.CodeIdempotencyConflict) {
		return fmt.Errorf("expected idempotency_conflict, got %s", failure.Code)
	}
	entity, ok := failure.Details["entity"].(map[string]any)
	if !ok {
		return fmt.Errorf("conflict details miss the entity reference: %v", failure.Details)
	}
	if entity["kind"] != entityKind || entity["id"] != entityID {
		return fmt.Errorf("conflict entity %v does not match %s %s", entity, entityKind, entityID)
	}
	original, ok := failure.Details["original_request"].(map[string]any)
	if !ok {
		return fmt.Errorf("conflict details miss the original request summary: %v", failure.Details)
	}
	if original["digest"] != digest {
		return fmt.Errorf("conflict digest %v does not match the recorded fingerprint", original["digest"])
	}
	if original["recorded_at"] != recordedAt {
		return fmt.Errorf("conflict recorded_at %v does not match %s", original["recorded_at"], recordedAt)
	}
	fields, ok := failure.Details["differing_fields"].([]any)
	if !ok {
		return fmt.Errorf("conflict details miss differing fields: %v", failure.Details)
	}
	if len(fields) != len(differing) {
		return fmt.Errorf("conflict differing fields %v do not match %v", fields, differing)
	}
	for index, name := range differing {
		if fields[index] != name {
			return fmt.Errorf("conflict differing fields %v do not match %v", fields, differing)
		}
	}
	return nil
}

func ensure(condition bool, message string) error {
	if condition {
		return nil
	}
	return errors.New(message)
}
