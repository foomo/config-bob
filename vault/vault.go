package vault

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const vaultAddr = "127.0.0.1:8200"

type readResponse struct {
	Data map[string]string
}

// GetVaultVersion get the version of the installed vault command line program
func GetVaultVersion() (version string, err error) {
	out, err := exec.Command("vault", "-v").Output()
	if err != nil {
		return "", errors.New("can not find vault")
	}
	return strings.Trim(string(out), "\n"), nil
}

// CheckEnv checks if vault is installed in the right version and set up
func CheckEnv() bool {
	return false
}

// ReadSecret data from a vault - env vars need to be set
func ReadSecret(path string) (secret map[string]string, err error) {
	if len(os.Getenv("VAULT_TOKEN")) == 0 {
		return nil, errors.New("VAULT_TOKEN is missing in env - can not call vault and ask for secrets")
	}
	// curl -v  -H "X-Vault-Token: $VAULT_TOKEN" $VAULT_ADDR/v1/secret/schild/smtp
	response, err := CallVault("/v1/" + path)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("could not get secret " + path + " : " + response.Status)
	}
	jsonBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	response.Body.Close()
	readResponse := &readResponse{}
	jsonErr := json.Unmarshal(jsonBytes, &readResponse)
	if jsonErr != nil {
		return nil, jsonErr
	}
	return readResponse.Data, nil
}

func CallVault(path string) (response *http.Response, err error) {
	addr := os.Getenv("VAULT_ADDR")
	if len(addr) == 0 {
		return nil, errors.New("VAULT_ADDR missing in env - can not call vault")
	}
	token := os.Getenv("VAULT_TOKEN")
	request, err := http.NewRequest("GET", addr+path, nil)

	if err != nil {
		return nil, err
	}
	request.Header.Add("X-Vault-Token", token)
	return http.DefaultClient.Do(request)
}
