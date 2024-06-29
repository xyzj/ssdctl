package model

import (
	"github.com/xyzj/gopsu/json"
)

type ToDo struct {
	Do     Jobs     `json:"do"`
	Name   string   `json:"name"`
	Exec   string   `json:"exec,omitempty"`
	Params []string `json:"params,omitempty"`
}

func (td *ToDo) ToJSON() []byte {
	b, err := json.Marshal(td)
	if err != nil {
		println("command marshal error:" + err.Error())
		return []byte{10}
	}
	b = append(b, 10)
	return b
}

func (td *ToDo) FromJSON(b []byte) {
	err := json.Unmarshal(b, td)
	if err != nil {
		println("command unmarshal error:" + err.Error())
	}
}

type ServiceParams struct {
	Pid        int      `yaml:"-"`
	Params     []string `yaml:"params"`
	Replace    []string `yaml:"replace,omitempty"`
	Env        []string `yaml:"env,omitempty"`
	Exec       string   `yaml:"exec"`
	Dir        string   `yaml:"dir,omitempty"`
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
)
