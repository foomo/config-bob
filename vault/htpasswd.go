package vault

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path"
	"runtime"

	"github.com/foomo/htpasswd"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v2"
)

// HtpasswdConfig config for htpasswd files
type HtpasswdConfig map[string][]string

// WriteHtpasswdFiles write them to files or update them
func WriteHtpasswdFiles(configFile string, hashAlgorithm htpasswd.HashAlgorithm) (err error) {
	config, err := ReadHtpasswdConfigFromFile(configFile)
	if err != nil {
		return
	}
	return writeHtpasswdFiles(config, hashAlgorithm)
}

func writeHtpasswdFiles(config HtpasswdConfig, hashAlgorithm htpasswd.HashAlgorithm) (err error) {
	for passwordFile, passwords := range config {
		// make sure that directories are there
		p := path.Dir(passwordFile)
		err = os.MkdirAll(p, 0o777)
		if err != nil {
			return errors.New("could not create path: " + p + " for file: " + passwordFile)
		}
		fmt.Println("updating passwords in:", passwordFile)
		hashedPasswords := htpasswd.HashedPasswords{}
		if _, statErr := os.Stat(passwordFile); statErr == nil {
			hashedPasswords, err = htpasswd.ParseHtpasswdFile(passwordFile)
			if err != nil {
				return fmt.Errorf("could not parse %q got error %q", passwordFile, err)
			}
		}
		// hash in parallel, bcrypt dominates, then apply in config order so the last entry for a user wins
		entries := make([]htpasswd.HashedPasswords, len(passwords))
		users := make([]string, len(passwords))
		var g errgroup.Group
		g.SetLimit(runtime.NumCPU())
		for i, passwordVaultPath := range passwords {
			g.Go(func() error {
				secret, err := Read(passwordVaultPath)
				if err != nil {
					return fmt.Errorf("could not read secret for path %q got error:: %q", passwordVaultPath, err)
				}
				user, userOk := secret["user"]
				password, passwordOk := secret["password"]
				if !userOk {
					return fmt.Errorf("secret from path %q is missing key user", passwordVaultPath)
				}
				if !passwordOk {
					return fmt.Errorf("secret from path %q is missing key password", passwordVaultPath)
				}
				entries[i] = htpasswd.HashedPasswords{}
				if err := entries[i].SetPassword(user, password, hashAlgorithm); err != nil {
					return fmt.Errorf("could not set password for %q in file %q got error %q", user, passwordFile, err)
				}
				users[i] = user
				return nil
			})
		}
		if err = g.Wait(); err != nil {
			return err
		}
		for i, passwordVaultPath := range passwords {
			fmt.Println("	", passwordVaultPath, ":", users[i])
			maps.Copy(hashedPasswords, entries[i])
		}
		// one write per file instead of one rewrite per user
		if err = hashedPasswords.WriteToFile(passwordFile); err != nil {
			return fmt.Errorf("could not write %q got error %q", passwordFile, err)
		}
	}
	return
}

// ReadHtpasswdConfigFromFile read htpasswd config from a file
func ReadHtpasswdConfigFromFile(filename string) (config HtpasswdConfig, err error) {
	configBytes, err := os.ReadFile(filename)
	if err != nil {
		return
	}
	return parseHtpasswdConfig(configBytes)
}

func parseHtpasswdConfig(configBytes []byte) (config HtpasswdConfig, err error) {
	config = HtpasswdConfig(make(map[string][]string))
	return config, yaml.Unmarshal(configBytes, config)
}
