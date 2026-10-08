//go:build unix

package vault

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeVault puts a vault script running body first on PATH
func fakeVault(t *testing.T, body string) {
	t.Helper()
	bin := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(bin, "vault"), []byte("#!/bin/sh\n"+body+"\n"), 0o755))
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("VAULT_ADDR", "http://127.0.0.1:1")
}

func TestLocalStartVaultExits(t *testing.T) {
	fakeVault(t, "exit 1")
	_, _, err := LocalStart(t.TempDir())
	require.ErrorContains(t, err, "exited before it was ready")
}

func TestLocalStartTimeout(t *testing.T) {
	fakeVault(t, "exec sleep 5")
	defer func(old time.Duration) { localStartTimeout = old }(localStartTimeout)
	localStartTimeout = 100 * time.Millisecond
	_, _, err := LocalStart(t.TempDir())
	require.ErrorContains(t, err, "did not start within")
}
