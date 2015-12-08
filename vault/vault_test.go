package vault

import "testing"

func TestVaultVersion(t *testing.T) {
	t.Log("what is the right fixture for that ?!")
	version, err := GetVaultVersion()
	if err != nil {
		t.Fatal("looks like vault is not installed or not in path")
	}
	if len(version) < 1 {
		t.Fatal("that version is very fishy")
	}
}
