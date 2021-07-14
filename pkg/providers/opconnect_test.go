package providers

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	onePasswordConnectTokenENV = "OP_CONNECT_TOKEN"
	onePasswordHost            = "http://localhost:8080"
	onePasswordVault           = "Globus Infrastructure"
)

func checkOnePasswordENV(t *testing.T) {
	if os.Getenv(onePasswordConnectTokenENV) == "" {
		t.Skip(onePasswordConnectTokenENV, " was not set, skipping integration tests")
	}
}

func TestNewOnePasswordProvider(t *testing.T) {
	checkOnePasswordENV(t)

	onePasswordToken := os.Getenv(onePasswordConnectTokenENV)

	provider, err := NewOnePasswordConnectProvider(onePasswordHost, onePasswordToken, onePasswordVault)
	require.NoError(t, err)
	require.NotNil(t, provider)

	token, err := provider.GetSecret("google-tracking-token.secret.token")
	require.NoError(t, err)
	require.Equal(t, "123456", token)

	email, err := provider.GetSecret("google-tracking-token.secret.email")
	require.NoError(t, err)
	require.Equal(t, "test@test.com", email)

	password, err := provider.GetSecret("google-tracking-token.secret.password")
	require.NoError(t, err)
	require.Equal(t, "Fv2.4!oJBTB7AGgnXa3ut_.7pURvPFFm", password)
}
