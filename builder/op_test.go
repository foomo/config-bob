//go:build unix

package builder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOnePassword(t *testing.T) {
	dir := t.TempDir()
	// the fake op echoes its arguments, so the test pins the 1Password CLI v2 syntax
	script := "#!/bin/sh\n[ \"$1\" = item ] || exit 1\necho \"$@\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "op"), []byte(script), 0o755))
	t.Setenv("PATH", dir)

	v, err := onePassword("my-item", "password")
	require.NoError(t, err)
	require.Equal(t, "item get my-item --fields password --reveal", v)

	out, err := process("op.txt", `user: {{ op "other" "user" }};`, nil)
	require.NoError(t, err)
	require.Equal(t, "user: item get other --fields user --reveal;", string(out), "the trailing newline of op must not leak into templates")

	t.Setenv("PATH", t.TempDir())
	_, err = onePassword("my-item", "password")
	require.Error(t, err)
}
