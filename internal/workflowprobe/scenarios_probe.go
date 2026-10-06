package workflowprobe

import (
	"net/http"

	"solo-0014-zone-rollout-control/internal/domain"
)

func runProbeIngestion() error {
	harness, err := newHarness()
	if err != nil {
		return err
	}
	defer harness.close()

	if _, err := enrollNode(harness, "resolver-probe-01", "central"); err != nil {
		return err
	}
	plan, revision, err := createActivePlan(harness, "probe.example", 2049, 1, 1, 50)
	if err != nil {
		return err
	}
	group, err := allocate(harness, plan.ID, "probe-lease-0001", "central", 1)
	if err != nil {
		return err
	}
	leaseID := group.LeaseIDs[0]
	first, err := submitProbe(harness, leaseID, "probe-request-0001", true, revision.Serial, 12)
	if err != nil {
		return err
	}
	if !first.Success || first.ObservedSerial != revision.Serial {
		return ensure(false, "probe values were not preserved")
	}
	replay, err := submitProbe(harness, leaseID, "probe-request-0001", true, revision.Serial, 12)
	if err != nil {
		return err
	}
	if replay.ID != first.ID {
		return ensure(false, "probe replay returned a different item")
	}
	_, err = submitProbe(harness, leaseID, "probe-request-0001", true, revision.Serial, 25)
	details, err := failureDetails(err, http.StatusConflict, string(domain.CodeIdempotencyConflict))
	if err != nil {
		return err
	}
	if err := checkConflictDetails(
		details,
		"probe-request-0001",
		"probe",
		first.ID,
		first.PayloadFingerprint,
		[]string{"latency_ms"},
	); err != nil {
		return err
	}
	replayed, err := submitProbe(harness, leaseID, "probe-request-0001", true, revision.Serial, 12)
	if err != nil {
		return err
	}
	if replayed.ID != first.ID {
		return ensure(false, "probe replay after conflict returned a different item")
	}
	err = harness.call(http.MethodPatch, "/v1/leases/"+leaseID+"/release", nil, nil)
	return expectFailure(err, http.StatusConflict, string(domain.CodeInvalidState))
}
