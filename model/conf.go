package model

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/xyzj/toolbox/pathtool"
	"gopkg.in/yaml.v3"
)

type Config struct {
	locker sync.RWMutex
	data   map[string]*ServiceParams
	cnfdir string
	piddir string
}

func cloneServiceParams(src *ServiceParams) *ServiceParams {
	if src == nil {
		return nil
	}
	dst := *src
	dst.Params = append([]string(nil), src.Params...)
	dst.Replace = append([]string(nil), src.Replace...)
	dst.Env = append([]string(nil), src.Env...)
	return &dst
}

func NewCnf(cnf, pid string) *Config {
	return &Config{
		locker: sync.RWMutex{},
		data:   make(map[string]*ServiceParams),
		cnfdir: cnf,
		piddir: pid,
	}
}

func (c *Config) ensureDefault(svr *ServiceParams) *ServiceParams {
	if !filepath.IsAbs(svr.Exec) {
		x, err := filepath.Abs(svr.Exec)
		if err == nil {
			svr.Exec = x
		}
	}
	if svr.Priority == 0 {
		svr.Priority = 200
	}
	svr.Priority = min(max(svr.Priority, 1), 255)
	svr.StartSec = max(svr.StartSec, 2)
	return svr
}

func (c *Config) Len() int {
	c.locker.RLock()
	defer c.locker.RUnlock()
	return len(c.data)
}

func (c *Config) FromFiles() {
	c.locker.Lock()
	defer c.locker.Unlock()
	fsd, err := os.ReadDir(c.cnfdir)
	if err != nil {
		println(c.cnfdir + " - " + err.Error())
		return
	}
	if c.data == nil {
		c.data = make(map[string]*ServiceParams)
	} else {
		for k := range c.data {
			delete(c.data, k)
		}
	}
	for _, fs := range fsd {
		if fs.IsDir() {
			continue
		}
		if filepath.Ext(fs.Name()) != ".yaml" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(c.cnfdir, fs.Name()))
		if err != nil {
			println(fs.Name() + " - " + err.Error())
			continue
		}
		s := &ServiceParams{}
		err = yaml.Unmarshal(b, s)
		if err != nil {
			println(fs.Name() + " - " + err.Error())
			continue
		}
		svrname := strings.TrimSuffix(fs.Name(), ".yaml")
		if b, err := os.ReadFile(filepath.Join(c.piddir, svrname+".pid")); err == nil {
			pidstr := strings.TrimSpace(string(b))
			pid, _ := strconv.Atoi(pidstr)
			s.Pid = pid
		}
		c.data[svrname] = c.ensureDefault(s)
	}
}

func (c *Config) AddItem(name string, svr *ServiceParams) error {
	c.locker.Lock()
	defer c.locker.Unlock()
	_, ok := c.data[name]
	if ok {
		return errors.New("service " + name + " already exist")
	}
	s := c.ensureDefault(cloneServiceParams(svr))
	b, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(c.cnfdir, name+".yaml"), b, 0o664)
	if err != nil {
		return err
	}
	c.data[name] = s
	return nil
}

func (c *Config) DelItem(name string) error {
	c.locker.Lock()
	defer c.locker.Unlock()
	if _, ok := c.data[name]; !ok {
		return errors.New("service " + name + " not exist")
	}
	delete(c.data, name)
	err := os.Remove(filepath.Join(c.cnfdir, name+".yaml"))
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
	if !ok {
		return nil, false
	}
	return cloneServiceParams(s), true
}

func (c *Config) SetRuntime(name string, pid int, manualStop bool) error {
	c.locker.Lock()
	defer c.locker.Unlock()
	s, ok := c.data[name]
	if !ok {
		return errors.New("service " + name + " not found")
	}
	s.Pid = pid
	s.ManualStop = manualStop
	return nil
}
func (c *Config) SetLevel(name string, l uint32) error {
	c.locker.Lock()
	defer c.locker.Unlock()
	s, ok := c.data[name]
	if !ok {
		return errors.New("service " + name + " not found")
	}
	s.Priority = max(min(l, 99), 1)
	b, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(c.cnfdir, name+".yaml"), b, 0o664)
}

func (c *Config) SetEnable(name string, enable bool) error {
	c.locker.Lock()
	defer c.locker.Unlock()
	s, ok := c.data[name]
	if !ok {
		return errors.New("service " + name + " not found")
	}
	if s.Enable == enable {
		return nil
	}
	s.Enable = enable
	b, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(c.cnfdir, name+".yaml"), b, 0o664)
}

func (c *Config) ForEach(f func(key string, value *ServiceParams) bool) {
	c.locker.RLock()
	ss := make([]*ServiceParams, 0, len(c.data))
	for k, v := range c.data {
		x := cloneServiceParams(v)
		x.name = k
		ss = append(ss, x)
	}
	c.locker.RUnlock()
	sort.Slice(ss, func(i, j int) bool {
		if ss[i].Priority == ss[j].Priority {
			return ss[i].name < ss[j].name
		}
		return ss[i].Priority < ss[j].Priority
	})
	for _, s := range ss {
		if !f(s.name, s) {
			break
		}
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
		sp := filepath.Join(c.cnfdir, k+".yaml")
		if pathtool.IsExist(sp) {
			continue
		}
		b, err := yaml.Marshal(c.ensureDefault(v))
		if err != nil {
			continue
		}
		os.WriteFile(sp, b, 0o664)
	}
	os.Remove(filepath.Join(pathtool.GetExecDir(), "ssdctld.yaml"))
}
