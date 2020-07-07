package builder

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func isOnePassworwordAvailable() bool {
	foundSession := false
	for _, kv := range os.Environ() {
		parts := strings.Split(kv, "=")
		if len(parts) > 0 && strings.HasPrefix(parts[0], "OP_SESSION_") {
			foundSession = true
			break
		}
	}
	return foundSession
}

func TestOnePassword(t *testing.T) {
	if !isOnePassworwordAvailable() {
		t.Skip("no op session found")
	}
	v, err := onePassword("kkwcxma7pbf3xaar7wgboj5zgm", "foo")
	assert.NoError(t, err)
	assert.Equal(t, "bar", v)
	v, err = onePassword("kkwcxma7pbf3xaar7wgboj5zgmsss", "foo")
	assert.Error(t, err)
	assert.Empty(t, v)
}
