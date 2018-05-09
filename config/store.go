package config

const (
	localStoreLocation = ".cfb/vault-store"
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
