package vault

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"

	"strings"
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

func parseHtpasswd(htpasswdBytes []byte) (passwords map[string]string, err error) {
	lines := strings.Split("\n", string(htpasswdBytes))
	passwords = make(map[string]string)
	for _, line := range lines {
		// scan lines
		line = strings.Trim(line, " ")
		if len(line) == 0 {
			// skipping empty lines
			continue
		}
		parts := strings.Split(":", line)
		for i, part := range parts {
			parts[i] = strings.Trim(part, " ")
		}
	}
	return
}
