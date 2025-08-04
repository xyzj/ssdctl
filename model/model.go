package model

import (
	"encoding/json"
)

type ToDo struct {
	Name   string   `json:"name"`
	Exec   string   `json:"exec,omitempty"`
	Params []string `json:"params,omitempty"`
	Do     Jobs     `json:"do"`
}

func (td *ToDo) ToJSON() []byte {
	b, err := json.Marshal(td)
	if err != nil {
		println("command marshal error:" + err.Error())
		return []byte{0}
	}
	b = append(b, 0)
	return b
}

func (td *ToDo) FromJSON(b []byte) error {
	err := json.Unmarshal(b, td)
	if err != nil {
		println("command unmarshal error:" + err.Error())
		return err
	}
	return nil
}

type ServiceParams struct {
	name       string   `yaml:"-"`
	Exec       string   `yaml:"exec"`
	Dir        string   `yaml:"dir,omitempty"`
	Params     []string `yaml:"params"`
	Replace    []string `yaml:"replace,omitempty"`
	Env        []string `yaml:"env,omitempty"`
	Pid        int      `yaml:"-"`
	StartSec   uint32   `yaml:"startsec"`
	Priority   uint32   `yaml:"priority"`
	Log2file   bool     `yaml:"log2file,omitempty"`
	Enable     bool     `yaml:"enable"`
	ManualStop bool     `yaml:"-"`
}

type Jobs byte

const (
	JobClose Jobs = iota
	JobStart
	JobStop
	JobRestart
	JobStatus
	JobEnable
	JobDisable
	JobCreate
	JobRemove
	JobList
	JobUpate
	JobShutdown
	JobSetLevel
)
