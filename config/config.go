package config

import (
	"errors"
	"io/ioutil"
	"os"

	env "github.com/Netflix/go-env"
	"gopkg.in/yaml.v2"
)

const CFG_ENV = "config"

func Load(out interface{}) error {
	if cfgStr := os.Getenv(CFG_ENV); cfgStr != "" {
		return LoadString(cfgStr, out)
	} else if len(os.Args) > 1 {
		return LoadFile(os.Args[1], out)
	} else {
		return errors.New("Not find any configuration info")
	}
}

func LoadFile(file string, out interface{}) error {
	conf, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	return loadData(conf, out)
}

func LoadString(str string, out interface{}) error {
	return loadData([]byte(str), out)
}

func loadData(data []byte, out interface{}) error {
	if err := yaml.Unmarshal(data, out); err != nil {
		return err
	}

	if _, err := env.UnmarshalFromEnviron(out); err != nil {
		return err
	}

	return nil
}
