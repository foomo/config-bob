package providers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type mockProvider struct {
}

func (m mockProvider) GetSecret(path string) (string, error) {
	return path, nil
}

func TestSecretProviderManager(t *testing.T) {
	manager := NewSecretProviderManager()

	err := manager.Register("mock", mockProvider{})
	require.NoError(t, err)

	// duplicate registration, throw error
	err = manager.Register("mock", mockProvider{})
	require.Error(t, err)

	value, err := manager.GetSecret("mock", "path.to.secret")
	require.NoError(t, err)
	require.Equal(t, "path.to.secret", value)

	value, err = manager.GetSecret("unknown", "path.to.secret")
	require.Error(t, err)
	require.Equal(t, "", value)
}
