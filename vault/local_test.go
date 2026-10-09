package vault

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalUnseal(t *testing.T) {
	var gotKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/v1/sys/unseal", r.URL.Path)
		body := map[string]string{}
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		gotKey = body["key"]
		_, _ = w.Write([]byte(`{"sealed":false,"t":1,"progress":0}`))
	}))
	defer server.Close()
	t.Setenv("VAULT_ADDR", server.URL)

	sealed, err := LocalUnseal("key-1")
	require.NoError(t, err)
	require.False(t, sealed)
	require.Equal(t, "key-1", gotKey)
}

func TestLocalUnsealError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"errors":["invalid key"]}`, http.StatusBadRequest)
	}))
	defer server.Close()
	t.Setenv("VAULT_ADDR", server.URL)

	_, err := LocalUnseal("bad-key")
	require.ErrorContains(t, err, "invalid key")
}

func TestFreeLoopbackAddrs(t *testing.T) {
	addr, clusterAddr, err := freeLoopbackAddrs()
	require.NoError(t, err)
	require.NotEqual(t, addr, clusterAddr)
	require.True(t, strings.HasPrefix(addr, "127.0.0.1:"), addr)
}

func TestLocalStartWithoutVault(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, _, err := LocalStart(t.TempDir())
	require.Error(t, err)
}
