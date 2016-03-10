package vault

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/foomo/htpasswd"

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
		err = os.MkdirAll(p, 0777)
		if err != nil {
			return errors.New("could not create path: " + p + " for file: " + passwordFile)
		}
		fmt.Println("updating passwords in:", passwordFile)
		for _, passwordVaultPath := range passwords {
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
			fmt.Println("	", passwordVaultPath, ":", user)
			err = htpasswd.SetPassword(passwordFile, user, password, hashAlgorithm)
			if err != nil {
				return fmt.Errorf("could not set password for %q in file %q got error %q", user, passwordFile, err)
			}
		}
	}
	return
}

// ReadHtpasswdConfigFromFile read htpasswd config from a file
func ReadHtpasswdConfigFromFile(filename string) (config HtpasswdConfig, err error) {
	configBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	return parseHtpasswdConfig(configBytes)
}

func parseHtpasswdConfig(configBytes []byte) (config HtpasswdConfig, err error) {
	config = HtpasswdConfig(make(map[string][]string))
	return config, yaml.Unmarshal(configBytes, config)
}
