package main

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// commands call os.Exit, so the test binary re-runs itself with these args as the CLI
const argsEnv = "CONFIG_BOB_TEST_ARGS"

func TestMain(m *testing.M) {
	if args, ok := os.LookupEnv(argsEnv); ok {
		os.Args = append([]string{"config-bob"}, strings.Fields(args)...)
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func runCLI(t *testing.T, args string) (string, int) {
	t.Helper()
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), argsEnv+"="+args)
	out, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return string(out), exitErr.ExitCode()
	}
	require.NoError(t, err)
	return string(out), 0
}

func TestBuildWithoutArgsPrintsUsage(t *testing.T) {
	out, code := runCLI(t, "build")
	require.Equal(t, 1, code, out)
	require.Contains(t, out, "usage:")
}

// throwaway credentials of the test vault in example/vault, also used by the Makefile
const (
	testVaultKey   = "ep3ipa04QViYX0POAQmz0+y9tpQLKPD8jOkjWa7um50="
	testVaultToken = "config-bob-test"
)

func TestBuildWithTestVault(t *testing.T) {
	if _, err := exec.LookPath("vault"); err != nil {
		t.Skip("vault binary not in PATH, run mise install")
	}
	t.Setenv("CFB_KEYS", testVaultKey)
	t.Setenv("CFB_TOKEN", testVaultToken)
	before := readTree(t, "example/vault")
	target := t.TempDir()

	out, code := runCLI(t, "build --vault-dir example/vault example/source-vault "+target)
	require.Equal(t, 0, code, out)
	rendered, err := os.ReadFile(filepath.Join(target, "app.yaml"))
	require.NoError(t, err)
	require.Equal(t, "user: bob\npassword: \"test-password\"\napiKey: test-api-key\n", string(rendered))
	require.Equal(t, before, readTree(t, "example/vault"), "the committed vault must stay untouched")
}

func readTree(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	require.NoError(t, filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		files[p] = string(b)
		return err
	}))
	return files
}

func TestVersionPrintsOnlyTheVersion(t *testing.T) {
	out, code := runCLI(t, "version")
	require.Equal(t, 0, code, out)
	require.Equal(t, "\n", out, "Version is empty in tests")
}
