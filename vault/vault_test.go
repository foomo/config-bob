package vault

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/foomo/htpasswd"
)

func poe(err error) {
	if err != nil {
		panic(err)
	}
}

func dummyVaultServer(handler func(r *http.Request) interface{}) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := handler(r)
		response := map[string]interface{}{
			"data": data,
		}
		responseBytes, err := json.Marshal(response)
		if err != nil {
			panic(err)
		}
		w.Write(responseBytes)
	}))
	os.Setenv("VAULT_TOKEN", "dummy-token")
	os.Setenv("VAULT_ADDR", ts.URL)
	return ts
}

func TestHtpasswd(t *testing.T) {
	ts := dummyVaultServer(func(r *http.Request) interface{} {
		response := map[string]string{
			"user":     "user-from" + r.URL.Path,
			"password": "dummy-password",
		}
		return response
	})
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

	cmd := exec.Command("tree", testDir)
	combined, err := cmd.CombinedOutput()
	t.Log("tree", err, string(combined))

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
		t.Fatal("looks like vault is not installed or not in path")
	}
	if len(version) < 1 {
		t.Fatal("that version is very fishy")
	}
}
