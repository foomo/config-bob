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
	Data map[string]interface{}
}

type Version struct {
	Major, Minor, Release int
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Release)
}

func (v Version) LowerThan(t Version) bool {
	if v.Major < t.Major {
		return true
	}
	if v.Major == t.Major && v.Minor < t.Minor {
		return true
	}
	if v.Major == t.Major && v.Minor == t.Minor && v.Release < t.Release {
		return true
	}
	return false
}

var vaultVersionCommand = exec.Command("vault", "-v")

func GetUnsealCommand(vaultKey string) (*exec.Cmd, error) {
	version, err := GetVaultVersionParsed()
	if err != nil {
		return nil, err
	}

	var args []string
	//https://www.vaultproject.io/guides/upgrading/upgrade-to-0.9.2.html#backwards-compatible-cli-changes
	//Breaking changes for 0.9.2+ => Operator

	if version.LowerThan(Version{Major: 0, Minor: 9, Release: 2}) {
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
func Read(path string) (secret map[string]interface{}, err error) {
	if Dummy {
		return map[string]interface{}{
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
