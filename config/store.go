package config

import (
	"os"
	"path"
)

const (
	defaultLocalStoreLocation = ".cfb/vault-store.json"
)

type KeyStore interface {
	Store(credentials VaultCredentials) error
	Lookup(path string) (credentials VaultCredentials, ok bool)
}

type VaultCredentials struct {
	Path  string   `json:"path"`
	Token string   `json:"token"`
	Keys  []string `json:"keys"`
}

func NewKeyStore() (KeyStore, error) {
	home, _ := os.LookupEnv("HOME")
	keyStorePath := path.Join(home, defaultLocalStoreLocation)
	return newLocalStore(keyStorePath)
}
