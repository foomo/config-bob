package vaultdummy

import (
	"testing"

	"github.com/foomo/config-bob/vault"
)

func TestDummy(t *testing.T) {
	t.Log("this is a dummy testing package")
	ts := DummyVaultServerSecretEcho()
	defer ts.Close()
	s, err := vault.ReadSecret("secret/hansi")
	if err != nil {
		t.Fatal(err)
	}
	u, ok := s["user"]
	if !ok || len(u) == 0 {
		t.Fatal("dummy is too dumb")
	}
}
