package config

const (
	defaultLocalStoreLocation = ".cfb/vault-store"
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
	return newLocalStore(defaultLocalStoreLocation)
}
