package config

import (
	"os"
	"io/ioutil"
	"encoding/json"
)

var _ KeyStore = &localStore{}

type localStore struct {
	path        string
	credentials []VaultCredentials
}

func newLocalStore(path string) (ls *localStore, err error) {
	ls = &localStore{path: path}
	err = ls.load()
	return
}

// Loads the store information from the pre-defined location
func (ls *localStore) load() error {
	if _, err := os.Stat(ls.path); os.IsNotExist(err) {
		return nil
	}

	data, err := ioutil.ReadFile(ls.path)
	if err != nil {
		return err
	}

	var loadedCredentials []VaultCredentials
	err = json.Unmarshal(data, &loadedCredentials)
	if err != nil {
		return err
	}

	ls.credentials = loadedCredentials
	return nil
}

func (ls *localStore) save() error {
	data, err := json.Marshal(ls.credentials)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(ls.path, data, 0644)
}

// Stores the credentials for the vault in the specified location
func (ls *localStore) Store(credentials VaultCredentials) error {
	newEntry := true
	for idx, cred := range ls.credentials {
		if cred.Path == credentials.Path {
			ls.credentials[idx] = credentials
			newEntry = false
			break
		}
	}
	if newEntry {
		ls.credentials = append(ls.credentials, credentials)
	}
	return ls.save()
}

// Looks up the credentials for the specified path
func (ls *localStore) Lookup(path string) (credentials VaultCredentials, ok bool) {
	for _, cred := range ls.credentials {
		if cred.Path == path {
			return cred, true
		}
	}
	return credentials, false
}
