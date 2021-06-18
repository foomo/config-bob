package templates

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_base64encode(t *testing.T) {
	data, err := base64encode("test")
	require.NoError(t, err)
	require.Equal(t, "dGVzdA==", data)
}
