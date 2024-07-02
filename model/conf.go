package model

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xyzj/gopsu/pathtool"
	"gopkg.in/yaml.v3"
)

type Config struct {
	locker sync.RWMutex
	data   map[string]*ServiceParams
	dir    string
}

func NewCnf(dir string) *Config {
	return &Config{
		locker: sync.RWMutex{},
		data:   make(map[string]*ServiceParams),
		dir:    dir,
	}
}

func (c *Config) FromFiles() {
	c.locker.Lock()
	defer c.locker.Unlock()
	if c.data == nil {
		c.data = make(map[string]*ServiceParams)
	} else {
		for k := range c.data {
			delete(c.data, k)
		}
	}
	fsd, err := os.ReadDir(c.dir)
	if err != nil {
		println(err.Error())
		return
	}
	for _, fs := range fsd {
		if fs.IsDir() {
			continue
		}
		if filepath.Ext(fs.Name()) != ".yaml" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(c.dir, fs.Name()))
		if err != nil {
			continue
		}
		s := &ServiceParams{}
		err = yaml.Unmarshal(b, s)
		if err != nil {
			continue
		}
		c.data[strings.ReplaceAll(fs.Name(), ".yaml", "")] = s
	}
}

func (c *Config) AddItem(name string, svr *ServiceParams) error {
	if !filepath.IsAbs(svr.Exec) {
		x, err := filepath.Abs(svr.Exec)
		if err == nil {
			svr.Exec = x
		}
	}
	if err := c.PutItem(name, svr); err != nil {
		return err
	}
	b, err := yaml.Marshal(svr)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(c.dir, name+".yaml"), b, 0o664)
}

func (c *Config) PutItem(name string, svr *ServiceParams) error {
	c.locker.Lock()
	defer c.locker.Unlock()
	_, ok := c.data[name]
	if ok {
		return fmt.Errorf("service " + name + " already exist")
	}
	c.data[name] = svr
	return nil
}

func (c *Config) DelItem(name string) error {
	c.locker.Lock()
	defer c.locker.Unlock()
	if _, ok := c.data[name]; !ok {
		return fmt.Errorf("service " + name + " not exist")
	}
	delete(c.data, name)
	err := os.Remove(filepath.Join(c.dir, name+".yaml"))
	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			return nil
		}
	}
	return err
}

func (c *Config) GetItem(name string) (*ServiceParams, bool) {
	c.locker.RLock()
	defer c.locker.RUnlock()
	s, ok := c.data[name]
	return s, ok
}

func (c *Config) SetEnable(name string, enable bool) error {
	c.locker.Lock()
	defer c.locker.Unlock()
	s, ok := c.data[name]
	if !ok {
		return fmt.Errorf("service " + name + " not found")
	}
	if s.Enable == enable {
		return nil
	}
	s.Enable = enable
	b, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(c.dir, name+".yaml"), b, 0o664)
}

func (c *Config) ForEach(f func(key string, value *ServiceParams) bool) {
	c.locker.RLock()
	defer c.locker.RUnlock()
	for k, v := range c.data {
		f(k, v)
	}
}

func (c *Config) Print() string {
	c.locker.RLock()
	defer c.locker.RUnlock()
	b, err := yaml.Marshal(c.data)
	if err != nil {
		return ""
	}
	return string(b)
}

func (c *Config) ConverFromOld() {
	b, err := os.ReadFile(filepath.Join(pathtool.GetExecDir(), "ssdctld.yaml"))
	if err != nil {
		return
	}

	a := make(map[string]*ServiceParams)
	err = yaml.Unmarshal(b, a)
	if err != nil {
		return
	}
	for k, v := range a {
		sp := filepath.Join(c.dir, k+".yaml")
		if pathtool.IsExist(sp) {
			continue
		}
		b, err := yaml.Marshal(v)
		if err != nil {
			continue
		}
		os.WriteFile(sp, b, 0o664)
	}
	os.Remove(filepath.Join(pathtool.GetExecDir(), "ssdctld.yaml"))
}
