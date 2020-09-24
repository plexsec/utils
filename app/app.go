package app

import (
	"fmt"
	"os"

	"github.com/plexsec/log"
	"github.com/plexsec/stat"
	"github.com/plexsec/utils/config"
)

type Config struct {
	MaxProcs uint32 `yaml:"max_procs"`

	Log  *log.Config  `yaml:"log"`
	Stat *stat.Config `yaml:"stat"`
}

type APP struct {
	initialized bool

	Name   string
	Config *Config
}

var instance *APP = &APP{}

func Instance() *APP {
	if instance.initialized {
		return instance
	} else {
		panic("APP is not initialized, call .New first")
	}

	return nil
}

func New(cfg interface{}) *APP {
	if instance.initialized {
		panic("APP initialized twice")
	}

	if len(os.Args) < 2 {
		panic(fmt.Errorf("Usage: %s CONFIG.yaml", os.Args[0]))
	}

	if err := config.LoadFile(os.Args[1], cfg); err != nil {
		panic(fmt.Errorf("Load config file error: %s", err))
	}

	instance.Config = &Config{}
	if err := config.LoadFile(os.Args[1], instance.Config); err != nil {
		panic(fmt.Errorf("Load  APP config error: %s", err))
	}

	instance.Name = os.Args[0]
	instance.initialized = true

	return instance
}

func (app *APP) InitLog() {
	log.Init(app.Config.Log)
}

func (app *APP) InitStat() {
	stat.Init(app.Config.Stat)
}

func (app *APP) WaitForever() {
	log.Info("Main thread going to wait forever")
	<-make(chan interface{})
}
