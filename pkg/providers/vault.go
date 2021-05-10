package providers

type Vault struct {
}

//TODO: Vault Provider
func NewVaultProvider() (*Vault, error) {
	return &Vault{}, nil
}

func (v *Vault) GetSecret(path string) (value string, err error) {
	panic("implement me")
}
