package cluster

import (
	"context"
	"testing"

	"github.com/coreeng/corectl/pkg/portal"
	"github.com/stretchr/testify/require"
)

func TestValidatePlanRejectsRegistrationGenerationChange(t *testing.T) {
	remote := portal.Cluster{ID: "one", Generation: "generation-one"}
	plan := portal.InstallationPlan{Operation: "install", ClusterID: "one", Generation: "generation-two"}

	err := validatePlan(remote, plan, "install")

	require.ErrorContains(t, err, "registration changed")
}

func TestUpgradeDoesNotReadUnusedReportBaseline(t *testing.T) {
	baselines, err := reportBaselines(context.Background(), nil, "one", "upgrade")

	require.NoError(t, err)
	require.Empty(t, baselines)
}

func TestLaterReportRequiresAValidStrictlyLaterTimestamp(t *testing.T) {
	require.True(t, laterReport("2026-08-29T10:00:01Z", "2026-08-29T10:00:00Z"))
	require.False(t, laterReport("2026-08-29T10:00:00Z", "2026-08-29T10:00:00Z"))
	require.False(t, laterReport("invalid", "2026-08-29T10:00:00Z"))
}
