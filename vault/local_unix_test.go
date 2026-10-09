//go:build unix

package vault

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// snapshot maps every file below dir to its content
func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	files := map[string]string{}
	require.NoError(t, filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		files[p] = string(b)
		return err
	}))
	return files
}

func TestLocalOpenCopyCleansUpWhenVaultFails(t *testing.T) {
	fakeVault(t, "exit 1")
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	folder := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(folder, "db", "core"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(folder, "db", "core", "_keyring"), []byte("x"), 0o644))

	_, err := LocalOpenCopy(folder, []string{"key"})
	require.ErrorContains(t, err, "exited before it was ready")
	entries, err := os.ReadDir(tmp)
	require.NoError(t, err)
	require.Empty(t, entries, "the temporary copy must be removed")
}

func TestLocalOpenCopyWithRealVault(t *testing.T) {
	if _, err := exec.LookPath("vault"); err != nil {
		t.Skip("vault binary not in PATH")
	}
	// initialise a throwaway vault with a single unseal key on its own random port
	folder := t.TempDir()
	addr, clusterAddr, err := freeLoopbackAddrs()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(folder, "db"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(folder, "config.hcl"), fmt.Appendf(nil, localCopyConfig, addr, clusterAddr), 0o644))
	t.Setenv("VAULT_ADDR", "http://"+addr)
	cmd, chanVaultErr, err := LocalStart(folder)
	require.NoError(t, err)
	request, err := http.NewRequest(http.MethodPut, "http://"+addr+"/v1/sys/init", strings.NewReader(`{"secret_shares":1,"secret_threshold":1}`))
	require.NoError(t, err)
	response, err := localClient.Do(request)
	require.NoError(t, err)
	initResult := struct {
		Keys []string `json:"keys"`
	}{}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&initResult))
	_ = response.Body.Close()
	require.Len(t, initResult.Keys, 1)
	require.NoError(t, cmd.Process.Kill())
	<-chanVaultErr
	before := snapshot(t, folder)

	// a listener on the classic port must not matter, the copy picks its own
	if blocker, err := net.Listen("tcp", vaultAddr); err == nil {
		defer blocker.Close()
	}

	_, err = LocalOpenCopy(folder, nil)
	require.ErrorContains(t, err, "still sealed")
	require.False(t, LocalIsRunning(), "a failed open must stop the vault")

	stop, err := LocalOpenCopy(folder, initResult.Keys)
	require.NoError(t, err)
	require.NotEqual(t, "http://"+vaultAddr, os.Getenv("VAULT_ADDR"))
	require.True(t, LocalIsRunning())
	stop()
	require.False(t, LocalIsRunning())
	require.Equal(t, before, snapshot(t, folder), "the vault folder must stay untouched")
}

func TestLocalStartTimeout(t *testing.T) {
	fakeVault(t, "exec sleep 5")
	defer func(old time.Duration) { localStartTimeout = old }(localStartTimeout)
	localStartTimeout = 100 * time.Millisecond
	_, _, err := LocalStart(t.TempDir())
	require.ErrorContains(t, err, "did not start within")
}
