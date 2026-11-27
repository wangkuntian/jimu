package actions

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunNewValidatesSelectors(t *testing.T) {
	_, err := RunNew(context.Background(), NewInput{Profile: "full", With: "user"})
	require.ErrorIs(t, err, ErrProfileWithMutuallyExclusive)

	_, err = RunNew(context.Background(), NewInput{Profile: "full", Shape: "app"})
	require.ErrorIs(t, err, ErrShapeRequiresWith)
}

func TestRunUpgradeValidatesModes(t *testing.T) {
	_, _, err := RunUpgrade(context.Background(), UpgradeInput{DryRun: true, Check: true})
	require.ErrorIs(t, err, ErrUpgradeModes)

	_, _, err = RunUpgrade(context.Background(), UpgradeInput{IncludeDiff: true})
	require.ErrorIs(t, err, ErrUpgradeDiagnosticModes)
}

func TestActionsHonorCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := RunNew(ctx, NewInput{})
	require.True(t, errors.Is(err, context.Canceled))
	_, err = RunCheck(ctx, CheckInput{})
	require.True(t, errors.Is(err, context.Canceled))
	_, err = RunAddCapability(ctx, AddInput{})
	require.True(t, errors.Is(err, context.Canceled))
	_, _, err = RunUpgrade(ctx, UpgradeInput{})
	require.True(t, errors.Is(err, context.Canceled))
}
