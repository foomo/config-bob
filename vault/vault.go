package vault

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const vaultAddr = "127.0.0.1:8200"

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

func GetSecret(path string) {
	// curl -v  -H "X-Vault-Token: $VAULT_TOKEN" $VAULT_ADDR/v1/secret/schild/smtp
}

func CallVault(path string) (response *http.Response, err error) {
	request, err := http.NewRequest("GET", os.Getenv("VAULT_ADDR")+path, nil)

	if err != nil {
		return nil, err
	}
	request.Header.Add("X-Vault-Token", os.Getenv("VAULT_TOKEN"))
	return http.DefaultClient.Do(request)
}

func StartAndInit(vaultFolder string) error {
	envVaultAddr := os.Getenv("VAULT_ADDR")
	if len(envVaultAddr) == 0 {
		os.Setenv("VAULT_ADDR", getLocalVaultAddress())
	}
	response, err := CallVault("/v1/sys/init")
	if err != nil {
		// vault not running
		return errors.New("vault not running: " + err.Error())
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return errors.New("could not read init response from vault :: " + err.Error())
	}
	response.Body.Close()

	fmt.Println(string(body))
	return nil
}
