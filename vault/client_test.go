package vault

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/foomo/htpasswd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// useVault points the shared client at handler and resets the secret cache
func useVault(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	t.Setenv("VAULT_ADDR", server.URL)
	Dummy = false
	client = sync.OnceValues(newClient)
	secretCache = map[string]map[string]string{}
	t.Cleanup(func() {
		client = sync.OnceValues(newClient)
		secretCache = map[string]map[string]string{}
	})
}

func respond(w http.ResponseWriter, data map[string]any) {
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
}

func TestReadCachesByPath(t *testing.T) {
	t.Setenv("VAULT_TOKEN", "test-token")
	var requests atomic.Int32
	useVault(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		assert.Equal(t, "/v1/secret/db", r.URL.Path)
		assert.Equal(t, "test-token", r.Header.Get("X-Vault-Token"))
		respond(w, map[string]any{"user": "app", "password": "pw"})
	})

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			secret, err := Read("secret/db")
			assert.NoError(t, err)
			assert.Equal(t, "pw", secret["password"])
		})
	}
	wg.Wait()
	secret, err := Read("secret/db")
	require.NoError(t, err)
	require.Equal(t, "app", secret["user"])
	require.Equal(t, int32(1), requests.Load())
}

func TestReadErrors(t *testing.T) {
	t.Setenv("VAULT_TOKEN", "test-token")
	useVault(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/secret/missing" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		respond(w, map[string]any{"port": 5432})
	})
	_, err := Read("secret/missing")
	require.ErrorContains(t, err, "no secret found")
	_, err = Read("secret/number")
	require.ErrorContains(t, err, "not a string")
}

func TestReadSanitizesPathAndNulls(t *testing.T) {
	t.Setenv("VAULT_TOKEN", "test-token")
	useVault(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/secret/db", r.URL.Path)
		respond(w, map[string]any{"user": "app", "comment": nil})
	})
	secret, err := Read("/secret/db/")
	require.NoError(t, err)
	require.Equal(t, map[string]string{"user": "app", "comment": ""}, secret)
}

func TestTree(t *testing.T) {
	t.Setenv("VAULT_TOKEN", "test-token")
	useVault(t, func(w http.ResponseWriter, r *http.Request) {
		listing := r.Method == "LIST" || r.URL.Query().Get("list") == "true"
		switch {
		case listing && r.URL.Path == "/v1/secret":
			respond(w, map[string]any{"keys": []string{"a", "dir/"}})
		case listing && r.URL.Path == "/v1/secret/dir":
			respond(w, map[string]any{"keys": []string{"b"}})
		case !listing:
			respond(w, map[string]any{"path": r.URL.Path})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	data, err := tree("secret/")
	require.NoError(t, err)
	_, err = tree("secret/empty")
	require.ErrorContains(t, err, "no entries found")
	require.Equal(t, map[string]map[string]string{
		"secret/a":     {"path": "/v1/secret/a"},
		"secret/dir/b": {"path": "/v1/secret/dir/b"},
	}, data)
}

func TestHtpasswdLastEntryWins(t *testing.T) {
	t.Setenv("VAULT_TOKEN", "test-token")
	useVault(t, func(w http.ResponseWriter, r *http.Request) {
		respond(w, map[string]any{"user": "admin", "password": "pw-" + filepath.Base(r.URL.Path)})
	})
	file := filepath.Join(t.TempDir(), "htpasswd")
	require.NoError(t, writeHtpasswdFiles(HtpasswdConfig{file: {"secret/a", "secret/b"}}, htpasswd.HashSHA))

	want := htpasswd.HashedPasswords{}
	require.NoError(t, want.SetPassword("admin", "pw-b", htpasswd.HashSHA))
	got, err := htpasswd.ParseHtpasswdFile(file)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestHtpasswdKeepsExistingUsers(t *testing.T) {
	Dummy = true
	t.Cleanup(func() { Dummy = false })
	file := filepath.Join(t.TempDir(), "htpasswd")
	existing := htpasswd.HashedPasswords{}
	require.NoError(t, existing.SetPassword("old-user", "old", htpasswd.HashSHA))
	require.NoError(t, existing.WriteToFile(file))

	require.NoError(t, writeHtpasswdFiles(HtpasswdConfig{file: {"secret/a", "secret/b"}}, htpasswd.HashSHA))
	passwords, err := htpasswd.ParseHtpasswdFile(file)
	require.NoError(t, err)
	require.Len(t, passwords, 3)
	require.Contains(t, passwords, "old-user")
	require.Contains(t, passwords, "user-fromsecret/b")
}
