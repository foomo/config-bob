package vault

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/mcuadros/go-version"
)

const vaultAddr = "127.0.0.1:8200"

type readResponse struct {
	Data map[string]string
}

var vaultVersionCommand = exec.Command("vault", "-v")

func GetUnsealCommand(vaultKey string) (*exec.Cmd, error) {
	vaultVersion, err := GetVaultVersionParsed()
	if err != nil {
		return nil, err
	}

	var args []string
	//https://www.vaultproject.io/guides/upgrading/upgrade-to-0.9.2.html#backwards-compatible-cli-changes
	//Breaking changes for 0.9.2+ => Operator

	if version.Compare(vaultVersion, "0.9.2", "<") {
		args = []string{"unseal", vaultKey}
	} else {
		args = []string{"operator", "unseal", vaultKey}
	}

	return exec.Command("vault", args...), nil
}

// GetVaultVersion get the version of the installed vault command line program
func GetVaultVersion() (version string, err error) {
	out, err := vaultVersionCommand.Output()
	if err != nil {
		return "", errors.New("can not find vault")
	}
	return strings.Trim(string(out), "\n"), nil
}

func GetVaultVersionParsed() (version string, err error) {
	versionString, err := GetVaultVersion()
	if err != nil {
		return
	}

	val := regexp.MustCompile(`Vault v(\d+\.\d+\.\d+)\s\('\w+'\)`).FindStringSubmatch(versionString)
	if len(val) != 2 {
		err = errors.New("invalid version format " + versionString)
	}

	return val[1], nil
}

func vaultErr(combinedOutput []byte, err error) error {
	return fmt.Errorf("err: %q, output: %q", err, string(combinedOutput))
}

// VaultDummy enables a built in dummy
var Dummy = false

// Read data from a vault - env vars need to be set
func Read(path string) (secret map[string]string, err error) {
	if Dummy {
		return map[string]string{
			"token":    "well-a-token",
			"name":     "call my name",
			"user":     "user-from" + path,
			"password": "dummy-password",
			"escape":   "muha\"haha",
		}, nil
	}
	jsonBytes, err := exec.Command("vault", "read", "-format", "json", path).CombinedOutput()
	if err != nil {
		return nil, vaultErr(jsonBytes, err)
	}
	response := &readResponse{}
	jsonErr := json.Unmarshal(jsonBytes, response)
	if jsonErr != nil {
		return nil, jsonErr
	}

	return response.Data, nil
}
