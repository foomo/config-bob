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
