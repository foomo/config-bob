package vault

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/foomo/htpasswd"
	"gopkg.in/yaml.v2"
	"os/exec"
)

func poe(err error) {
	if err != nil {
		panic(err)
	}
}

func TestHtpasswd(t *testing.T) {
	VaultDummy = true
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
		t.Skip("looks like vault is not installed or not in path")
		return
	}
	if len(version) < 1 {
		t.Fatal("that version is very fishy")
	}
}

func TestGetVaultVersionParsed(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		wantMajor   int
		wantMinor   int
		wantRelease int
		wantErr     bool
	}{
		{"standard", "Vault v0.9.5 ('36edb4d42380d89a897e7f633046423240b710d9')", 0, 9, 5, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vaultVersionCommand = exec.Command("echo", tt.version)
			gotMajor, gotMinor, gotRelease, err := GetVaultVersionParsed()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetVaultVersionParsed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotMajor != tt.wantMajor {
				t.Errorf("GetVaultVersionParsed() gotMajor = %v, want %v", gotMajor, tt.wantMajor)
			}
			if gotMinor != tt.wantMinor {
				t.Errorf("GetVaultVersionParsed() gotMinor = %v, want %v", gotMinor, tt.wantMinor)
			}
			if gotRelease != tt.wantRelease {
				t.Errorf("GetVaultVersionParsed() gotRelease = %v, want %v", gotRelease, tt.wantRelease)
			}
		})
	}
}
