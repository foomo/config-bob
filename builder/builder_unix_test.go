//go:build unix

package builder

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteProcessingResultIgnoresUmask(t *testing.T) {
	old := syscall.Umask(0o077)
	defer syscall.Umask(old)
	target := t.TempDir()
	writeOne(t, target, "shared", 0o644)
	info, err := os.Stat(filepath.Join(target, "out.conf"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), info.Mode().Perm())

	writeOne(t, target, "shared", 0o666)
	info, err = os.Stat(filepath.Join(target, "out.conf"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o644), info.Mode().Perm(), "group and others must never get write access")
}

func TestWriteProcessingResultFoldersAreTraversable(t *testing.T) {
	old := syscall.Umask(0o022)
	defer syscall.Umask(old)
	source := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(source, "httpd"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(source, "httpd", "test.conf"), []byte("x"), 0o644))
	r, err := processFolder(source, nil)
	require.NoError(t, err)
	target := filepath.Join(t.TempDir(), "out")
	require.NoError(t, WriteProcessingResult(target, r))
	for _, folder := range []string{target, filepath.Join(target, "httpd")} {
		info, err := os.Stat(folder)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o755), info.Mode().Perm(), folder)
	}
}
