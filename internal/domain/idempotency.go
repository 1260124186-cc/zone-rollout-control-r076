package domain

import "time"

// ConflictEntityRef identifies the object an idempotent request key is bound
// to. Only public API identifiers are exposed; internal index keys and audit
// identifiers never leave the server.
type ConflictEntityRef struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	PlanID  string `json:"plan_id,omitempty"`
	LeaseID string `json:"lease_id,omitempty"`
}

// RequestDigest is the stable summary of the original request recorded under
// an idempotency key. The digest is one-way, so the summary never reveals the
// original payload values.
type RequestDigest struct {
	Digest     string    `json:"digest"`
	RecordedAt time.Time `json:"recorded_at"`
}

// IdempotencyConflict builds the stable conflict error returned when a request
// key is reused with different content. The details describe the original
// request through its digest and name the key fields that differ, without
// echoing the stored payload.
func IdempotencyConflict(
	message string,
	requestKey string,
	entity ConflictEntityRef,
	original RequestDigest,
	differingFields []string,
) *DomainError {
	if differingFields == nil {
		differingFields = []string{}
	}
	return NewError(CodeIdempotencyConflict, message, map[string]any{
		"request_key":      requestKey,
		"entity":           entity,
		"original_request": original,
		"differing_fields": differingFields,
	})
}

// DiffLeaseFields names the lease request fields whose values differ from the
// recorded allocation group. The input must already be normalized.
func DiffLeaseFields(group LeaseGroup, input LeaseInput) []string {
	differing := make([]string, 0, 3)
	if group.Region != input.Region {
		differing = append(differing, "region")
	}
	if group.LeaseSeconds != input.LeaseSeconds {
		differing = append(differing, "lease_seconds")
	}
	if group.MaxNodes != input.MaxNodes {
		differing = append(differing, "max_nodes")
	}
	return differing
}

// DiffProbeFields names the probe request fields whose values differ from the
// stored probe.
func DiffProbeFields(existing Probe, input ProbeInput) []string {
	differing := make([]string, 0, 5)
	if existing.RequestKey != input.RequestKey {
		differing = append(differing, "request_key")
	}
	if existing.Success != input.Success {
		differing = append(differing, "success")
	}
	if existing.ObservedSerial != input.ObservedSerial {
		differing = append(differing, "observed_serial")
	}
	if existing.LatencyMS != input.LatencyMS {
		differing = append(differing, "latency_ms")
	}
	if !existing.ObservedAt.Equal(input.ObservedAt.UTC()) {
		differing = append(differing, "observed_at")
	}
	return differing
}

// DiffAssessmentFields names the evaluation context fields that changed since
// the request key was first used for an assessment.
func DiffAssessmentFields(existing RolloutAssessment, revisionID string, probeCount int) []string {
	differing := make([]string, 0, 2)
	if existing.RevisionID != revisionID {
		differing = append(differing, "revision_id")
	}
	if existing.ProbeCount != probeCount {
		differing = append(differing, "probes")
	}
	return differing
}
