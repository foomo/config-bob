package vault

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// ReadHtpasswdConfigFromFile read htpasswd config from a file
func ReadHtpasswdConfigFromFile(filename string) (config map[string][]string, err error) {
	configBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	return parseHtpasswdConfig(configBytes)
}

func parseHtpasswdConfig(configBytes []byte) (config map[string][]string, err error) {
	config = make(map[string][]string)
	return config, yaml.Unmarshal(configBytes, config)
}
