package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/stretchr/testify/require"
)

func TestCanvasTrustTokenLifecycle(t *testing.T) {
	common.RedisEnabled = false
	setting.CanvasTrustEnabled = true
	setting.CanvasTrustSecret = "test-secret"
	setting.CanvasTrustTokenTTL = 300

	token, err := CreateCanvasTrustToken(42)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	userID, err := consumeCanvasTrustToken(token)
	require.NoError(t, err)
	require.Equal(t, 42, userID)

	_, err = consumeCanvasTrustToken(token)
	require.ErrorIs(t, err, ErrCanvasTrustInvalid)
}

func TestVerifyCanvasTrustTokenRejectsEmptyToken(t *testing.T) {
	setting.CanvasTrustEnabled = true
	setting.CanvasTrustSecret = "test-secret"

	_, err := VerifyCanvasTrustToken(" ")
	require.ErrorIs(t, err, ErrCanvasTrustInvalid)
}
