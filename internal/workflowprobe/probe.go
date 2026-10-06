package workflowprobe

import (
	"errors"
	"fmt"
	"io"
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
	_, expectErr := failureDetails(err, status, code)
	return expectErr
}

func failureDetails(err error, status int, code string) (map[string]any, error) {
	var failure *apiFailure
	if !errors.As(err, &failure) {
		if err == nil {
			return nil, fmt.Errorf("expected HTTP %d %s but request succeeded", status, code)
		}
		return nil, fmt.Errorf("expected HTTP %d %s but got %v", status, code, err)
	}
	if failure.Status != status || failure.Code != code {
		return nil, fmt.Errorf("expected HTTP %d %s, got HTTP %d %s", status, code, failure.Status, failure.Code)
	}
	return failure.Details, nil
}

// checkConflictDetails verifies the stable idempotency conflict shape shared
// by every entry point: the reused request key, the associated entity
// reference, and an original request summary that exposes only the stored
// fingerprint and the differing field names, never payload values.
func checkConflictDetails(
	details map[string]any,
	requestKey string,
	entity string,
	id string,
	fingerprint string,
	differing []string,
) error {
	if details["request_key"] != requestKey {
		return fmt.Errorf("conflict request_key = %v, want %s", details["request_key"], requestKey)
	}
	if details["entity"] != entity {
		return fmt.Errorf("conflict entity = %v, want %s", details["entity"], entity)
	}
	if details["id"] != id {
		return fmt.Errorf("conflict id = %v, want %s", details["id"], id)
	}
	summary, ok := details["original_request"].(map[string]any)
	if !ok {
		return errors.New("conflict details are missing the original_request summary")
	}
	if len(summary) != 2 {
		return fmt.Errorf("original_request must expose only fingerprint and differing_fields, got %v", summary)
	}
	if summary["fingerprint"] != fingerprint {
		return fmt.Errorf("original_request fingerprint = %v, want %s", summary["fingerprint"], fingerprint)
	}
	fields, ok := summary["differing_fields"].([]any)
	if !ok {
		return fmt.Errorf("original_request differing_fields has unexpected shape %v", summary["differing_fields"])
	}
	if len(fields) != len(differing) {
		return fmt.Errorf("original_request differing_fields = %v, want %v", fields, differing)
	}
	for index, want := range differing {
		if fields[index] != want {
			return fmt.Errorf("original_request differing_fields = %v, want %v", fields, differing)
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
