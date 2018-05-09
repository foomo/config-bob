package config

import (
	"testing"
	"io/ioutil"
	"os"
	"github.com/stretchr/testify/assert"
)

func Test_LocalStore_Operations(t *testing.T) {
	file, err := ioutil.TempFile(os.TempDir(), "vault-store")
	assert.NoError(t, err)
	os.Remove(file.Name())
	defer os.Remove(file.Name())

	ls, err := newLocalStore(file.Name())
	assert.NoError(t, err)

	assert.NoError(t, ls.Store(VaultCredentials{"temp1", "token0", []string{"keys"}}))
	assert.NoError(t, ls.Store(VaultCredentials{"temp1", "token1", []string{"keys"}}))
	assert.NoError(t, ls.Store(VaultCredentials{"temp2", "token2", []string{"keys"}}))

	assert.Equal(t, 2, len(ls.credentials))

	vc, ok := ls.Lookup("temp1")
	assert.True(t, ok)
	assert.Equal(t, VaultCredentials{"temp1", "token1", []string{"keys"}}, vc)

	_, ok = ls.Lookup("fake")
	assert.False(t, ok)
}

func Test_LocalStore_Recovery(t *testing.T) {
	file, err := ioutil.TempFile(os.TempDir(), "vault-store")
	assert.NoError(t, err)
	os.Remove(file.Name())
	defer os.Remove(file.Name())

	ls, err := newLocalStore(file.Name())
	assert.NoError(t, err)

	creds := VaultCredentials{"test", "token", []string{"key"}}
	assert.NoError(t, ls.Store(creds))

	ls, err = newLocalStore(file.Name())
	assert.NoError(t, err)

	loadedCreds, ok := ls.Lookup("test")
	assert.True(t, ok)
	assert.Equal(t, creds, loadedCreds)
}
