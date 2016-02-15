package vault

import (
	"io/ioutil"
	"os"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/foomo/config-bob/vaultdummy"
	"github.com/foomo/htpasswd"
)

func poe(err error) {
	if err != nil {
		panic(err)
	}
}

func TestHtpasswd(t *testing.T) {
	ts := vaultdummy.DummyVaultServerSecretEcho()
	defer ts.Close()
	testDir, err := ioutil.TempDir(os.TempDir(), "htpasswd-config-test-dir-")
	poe(err)
	testConfigFile, err := ioutil.TempFile(os.TempDir(), "htpasswd-config")
	poe(err)

	cnf := map[string][]string{
		testDir + "/foo/test/bar": []string{
			"secret/foo",
			"secret/bar",
		},
		testDir + "/foo/hansi": []string{
			"secret/a",
		},
	}
	configBytes, err := yaml.Marshal(cnf)
	poe(err)
	poe(ioutil.WriteFile(testConfigFile.Name(), configBytes, 0600))
	poe(WriteHtpasswdFiles(testConfigFile.Name(), htpasswd.HashBCrypt))

	/*
		cmd := exec.Command("tree", testDir)
		combined, err := cmd.CombinedOutput()
		t.Log("tree", err, string(combined))
	*/
	for htpasswdFile, secretPaths := range cnf {
		passwords, err := htpasswd.ParseHtpasswdFile(htpasswdFile)
		//poe(err)
		if len(passwords) != len(secretPaths) {
			t.Fatal("wrong number of passwords in", htpasswdFile, passwords, err)
		}
	}

}

func TestVaultVersion(t *testing.T) {
	version, err := GetVaultVersion()
	if err != nil {
		t.Log("looks like vault is not installed or not in path")
	}
	if len(version) < 1 {
		t.Fatal("that version is very fishy")
	}
}
