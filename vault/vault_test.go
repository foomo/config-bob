package vault

import (
	"os"
	"testing"

	"github.com/foomo/htpasswd"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestHtpasswd(t *testing.T) {
	Dummy = true
	testDir, err := os.MkdirTemp(os.TempDir(), "htpasswd-config-test-dir-")
	require.NoError(t, err)
	testConfigFile, err := os.CreateTemp(os.TempDir(), "htpasswd-config")
	require.NoError(t, err)

	cnf := map[string][]string{
		testDir + "/foo/test/bar": {
			"secret/foo",
			"secret/bar",
		},
		testDir + "/foo/hansi": {
			"secret/a",
		},
	}
	configBytes, err := yaml.Marshal(cnf)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(testConfigFile.Name(), configBytes, 0o600))
	require.NoError(t, WriteHtpasswdFiles(testConfigFile.Name(), htpasswd.HashBCrypt))

	for htpasswdFile, secretPaths := range cnf {
		passwords, err := htpasswd.ParseHtpasswdFile(htpasswdFile)
		require.NoError(t, err)
		if len(passwords) != len(secretPaths) {
			t.Fatal("wrong number of passwords in", htpasswdFile, passwords, err)
		}
	}
}
