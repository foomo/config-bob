package main

import (
	"errors"
	"os"
	"os/exec"
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

func TestVersionPrintsOnlyTheVersion(t *testing.T) {
	out, code := runCLI(t, "version")
	require.Equal(t, 0, code, out)
	require.Equal(t, "\n", out, "Version is empty in tests")
}
