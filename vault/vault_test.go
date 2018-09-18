package vault

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"testing"

	"github.com/foomo/htpasswd"
	"gopkg.in/yaml.v2"
)

func poe(err error) {
	if err != nil {
		panic(err)
	}
}

func TestHtpasswd(t *testing.T) {
	Dummy = true
	testDir, err := ioutil.TempDir(os.TempDir(), "htpasswd-config-test-dir-")
	poe(err)
	testConfigFile, err := ioutil.TempFile(os.TempDir(), "htpasswd-config")
	poe(err)

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
		wantVersion string
		wantErr     bool
	}{
		{"standard", "Vault v0.9.5 ('36edb4d42380d89a897e7f633046423240b710d9')", "0.9.5", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vaultVersionCommand = exec.Command("echo", tt.version)
			version, err := GetVaultVersionParsed()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetVaultVersionParsed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if version != tt.wantVersion {
				t.Errorf("GetVaultVersionParsed() gotVersion = %v, want %v", version, tt.wantVersion)
			}
		})
	}
}

func TestGetUnsealCommand(t *testing.T) {
	vaultKey := "fake-key"
	vaultOperator := exec.Command("vault", "operator", "unseal", vaultKey)
	vaultDeprecated := exec.Command("vault", "unseal", vaultKey)

	tests := []struct {
		name    string
		version string
		want    *exec.Cmd
		wantErr bool
	}{
		{"deprecated", "0.9.0", vaultDeprecated, false},
		{"operator", "0.9.5", vaultOperator, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vaultVersionCommand = exec.Command("echo", fmt.Sprintf("Vault v%s ('36edb4d42380d89a897e7f633046423240b710d9')", tt.version))
			got, err := GetUnsealCommand(vaultKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUnsealCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetUnsealCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}
