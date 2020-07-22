package vault

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const vaultAddr = "127.0.0.1:8200"

type readResponse struct {
	Data map[string]string
}

type Version struct {
	Major, Minor, Release int
}

var vaultVersionCommand = exec.Command("vault", "-v")

func isVersionLower(source Version, target Version) bool {
	if source.Major < target.Major {
		return true
	}
	if source.Minor < target.Minor {
		return true
	}
	if source.Release < target.Release {
		return true
	}
	return false
}

func GetUnsealCommand(vaultKey string) (*exec.Cmd, error) {
	version, err := GetVaultVersionParsed()
	if err != nil {
		return nil, err
	}

	var args []string
	//https://www.vaultproject.io/guides/upgrading/upgrade-to-0.9.2.html#backwards-compatible-cli-changes
	//Breaking changes for 0.9.2+ => Operator
	if isVersionLower(version, Version{0, 9, 2}) {
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

func GetVaultVersionParsed() (version Version, err error) {
	versionString, err := GetVaultVersion()
	if err != nil {
		return
	}

	val := regexp.MustCompile(`Vault v(\d+).(\d+).(\d+).*`).FindStringSubmatch(versionString)
	if len(val) != 4 {
		return Version{}, fmt.Errorf("invalid version format %q", versionString)
	}

	versionData := make([]int, 3)
	for i := 0; i < 3; i++ {
		versionData[i], err = strconv.Atoi(val[i+1])
		if err != nil {
			return Version{}, err
		}
	}
	return Version{versionData[0], versionData[1], versionData[2]}, nil
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
