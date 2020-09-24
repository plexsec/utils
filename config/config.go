package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Load config from file
func LoadFile(file string, out interface{}) error {
	conf, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	return loadData(conf, out)
}

// Load config from string
func LoadString(str string, out interface{}) error {
	return loadData([]byte(str), out)
}

func loadData(data []byte, out interface{}) error {
	if err := yaml.Unmarshal(data, out); err != nil {
		return err
	}

	return nil
}
