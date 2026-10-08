package vault

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

const vaultAddr = "127.0.0.1:8200"

type readResponse struct {
	Data map[string]string
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
