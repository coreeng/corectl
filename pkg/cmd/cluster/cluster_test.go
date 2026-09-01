package cluster

import (
	"context"
	"testing"

	"github.com/coreeng/corectl/pkg/cmd/platformruntime"
	"github.com/coreeng/corectl/pkg/portal"
	"github.com/stretchr/testify/require"
)

func TestOperationCommandsAcceptOperationID(t *testing.T) {
	for _, operation := range []string{"install", "convert", "repair"} {
		t.Run(operation, func(t *testing.T) {
			cmd := operationCmd(&platformruntime.Runtime{}, operation)

			flag := cmd.Flags().Lookup("operation-id")

			require.NotNil(t, flag)
			require.Empty(t, flag.DefValue)
		})
	}
	require.Nil(t, operationCmd(&platformruntime.Runtime{}, "upgrade").Flags().Lookup("operation-id"))
}

func TestOperationCommandRejectsInvalidOperationID(t *testing.T) {
	cmd := operationCmd(&platformruntime.Runtime{}, "install")
	require.NoError(t, cmd.Flags().Set("operation-id", "not-a-uuid"))

	err := cmd.Args(cmd, []string{"one"})

	require.ErrorContains(t, err, "invalid --operation-id")
}

func TestCanonicalOperationIDNormalizesAcceptedUUIDForms(t *testing.T) {
	result, err := canonicalOperationID("7ab3d9591f824ab2ac55feb4d91c76a1")

	require.NoError(t, err)
	require.Equal(t, "7ab3d959-1f82-4ab2-ac55-feb4d91c76a1", result)
}

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
