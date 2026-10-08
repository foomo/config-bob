//go:build unix

package vault

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadTokenHelper(t *testing.T) {
	t.Setenv("VAULT_TOKEN", "")
	dir := t.TempDir()
	helper := filepath.Join(dir, "token-helper")
	require.NoError(t, os.WriteFile(helper, []byte("#!/bin/sh\necho helper-token\n"), 0o755))
	config := filepath.Join(dir, "vault.hcl")
	require.NoError(t, os.WriteFile(config, []byte("token_helper = \""+helper+"\"\n"), 0o600))
	t.Setenv("VAULT_CONFIG_PATH", config)
	useVault(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "helper-token", r.Header.Get("X-Vault-Token"))
		respond(w, map[string]any{"user": "app"})
	})
	_, err := Read("secret/db")
	require.NoError(t, err)
}
