// 使用unix socket实现的一个控制台管理程序，利用start-stop-daemon实现进程管理
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/mohae/deepcopy"
	"github.com/xyzj/gopsu/godaemon"
	"github.com/xyzj/gopsu/logger"
	"github.com/xyzj/gopsu/loopfunc"
	"github.com/xyzj/gopsu/tools"
	"github.com/xyzj/gopsu/yaml"
)

type serviceParams struct {
	Params     []string `yaml:"params"`
	Exec       string   `yaml:"exec"`
	Enable     bool     `yaml:"enable"`
	manualStop bool
}
type mapPS struct {
	locker   sync.RWMutex
	data     map[string]*serviceParams
	yamlfile *yaml.Config
}

func (m *mapPS) init() {
	m.locker.Lock()
	m.data = make(map[string]*serviceParams)
	// 默认启动的
	m.data["caddy"] = &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/caddy",
		Params: []string{"run"},
	}
	m.data["ttyd"] = &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/ttyd",
		Params: []string{"-p 6818", "-m 3", "bash"},
	}
	m.data["backend"] = &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/backend",
		Params: []string{"-portable", "-conf=backend.conf", "-http=6819", "-forcehttp=true"},
	}
	m.data["uas"] = &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/uas",
		Params: []string{"-portable", "-conf=luwak.conf", "-http=6820", "-forcehttp=false"},
	}
	m.data["ecms-mod"] = &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/ecms-mod",
		Params: []string{"-portable", "-conf=ecms.conf", "-http=6821", "-tcp=6828", "-tcpmodule=wlst", "-forcehttp=false"},
	}
	m.data["task"] = &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/task",
		Params: []string{"-portable", "-conf=luwak.conf", "-http=6822", "-forcehttp=false"},
	}
	m.data["event"] = &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/logger",
		Params: []string{"-portable", "-conf=luwak.conf", "-http=6823", "-forcehttp=false"},
	}
	m.data["adc"] = &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/adc",
		Params: []string{"-portable", "-conf=luwak.conf", "-http=6832", "-forcehttp=false"},
	}
	m.data["ccb"] = &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/ccb",
		Params: []string{"-portable", "-conf=luwak.conf", "-http=6836", "-forcehttp=false"},
	}
	m.data["alarm"] = &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/alarmlog",
		Params: []string{"-portable", "-conf=luwak.conf", "-http=6837", "-forcehttp=false"},
	}
	m.data["msgpush"] = &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/msgpush",
		Params: []string{"-portable", "-conf=luwak.conf", "-http=6824", "-forcehttp=false"},
	}
	m.data["bsjk"] = &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/businessjk",
		Params: []string{"-portable", "-conf=luwak.conf", "-http=6827", "-forcehttp=false"},
	}
	m.data["uiact"] = &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/netcore/userinteraction",
		Params: []string{"--log=/opt/bin/log/userinteraction", "--conf=/opt/bin/conf/userinteraction"},
	}
	m.data["dpwlst"] = &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/netcore/dataparser-wlst",
		Params: []string{"--log=/opt/bin/log/dataparser-wlst", "--conf=/opt/bin/conf/dataparser-wlst"},
	}
	m.data["dm"] = &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/netcore/datamaintenance",
		Params: []string{"--log=/opt/bin/log/datamaintenance", "--conf=/opt/bin/conf/datamaintenance"},
	}
	// 默认不起动的
	m.data["asset"] = &serviceParams{
		Enable: false,
		Exec:   "/opt/bin/assetmanager",
		Params: []string{"-portable", "-conf=luwak.conf", "-http=6825", "-forcehttp=false"},
	}
	m.data["gis"] = &serviceParams{
		Enable: false,
		Exec:   "/opt/bin/gismanager",
		Params: []string{"-portable", "-conf=luwak.conf", "-http=6826", "-forcehttp=false"},
	}
	m.data["nboam"] = &serviceParams{
		Enable: false,
		Exec:   "/opt/bin/nboam",
		Params: []string{"-portable", "-conf=nboam.conf", "-http=6835", "-forcehttp=false"},
	}
	m.data["ftpupg"] = &serviceParams{
		Enable: false,
		Exec:   "/opt/bin/ftpupgrade",
		Params: []string{"-portable", "-conf=ftp.conf", "-http=6829", "-ftp=6830", "-forcehttp=false"},
	}
	m.data["dpnb"] = &serviceParams{
		Enable: false,
		Exec:   "/opt/bin/netcore/dataparser-nbiot",
		Params: []string{"--log=/opt/bin/log/dataparser-nbiot", "--conf=/opt/bin/conf/dataparser-nbiot"},
	}
	m.data["sslrenew"] = &serviceParams{
		Enable: false,
		Exec:   "/opt/bin/sslrenew",
	}
	m.locker.Unlock()
}
func (m *mapPS) readfile() {
	m.locker.Lock()
	m.data = make(map[string]*serviceParams)
	m.yamlfile.Read(&m.data)
	m.locker.Unlock()
}
func (m *mapPS) savefile() {
	m.locker.Lock()
	m.yamlfile.Write(m.data)
	m.locker.Unlock()
}
func (m *mapPS) manualstop(key string, stop bool) {
	m.locker.Lock()
	_, ok := m.data[key]
	if ok {
		m.data[key].manualStop = stop
	}
	m.locker.Unlock()
}
func (m *mapPS) setenable(key string, en bool) {
	m.locker.Lock()
	_, ok := m.data[key]
	if ok {
		m.data[key].Enable = en
	}
	m.locker.Unlock()
}
func (m *mapPS) store(key, exec string, params ...string) {
	m.locker.Lock()
	m.data[key] = &serviceParams{
		Exec:   exec,
		Params: params,
		Enable: true,
	}
	m.locker.Unlock()
}
func (m *mapPS) delete(key string) {
	m.locker.Lock()
	delete(m.data, key)
	m.locker.Unlock()
}
func (m *mapPS) load(key string) (*serviceParams, bool) {
	m.locker.RLock()
	v, ok := m.data[key]
	m.locker.RUnlock()
	if ok {
		return v, true
	}
	return nil, false
}
func (m *mapPS) xrange(f func(key string, value *serviceParams) bool) {
	m.locker.RLock()
	x := deepcopy.Copy(m.data).(map[string]*serviceParams)
	m.locker.RUnlock()
	defer func() {
		recover()
	}()
	for k, v := range x {
		if !f(k, v) {
			break
		}
	}
}

var (
	stdlog   logger.Logger
	svrconf  = mapPS{locker: sync.RWMutex{}, data: make(map[string]*serviceParams), yamlfile: yaml.New(tools.JoinPathFromHere("extsvr.yaml"))}
	sendfmt  = `%20s|%s|`
	sigc     = make(chan os.Signal, 1)
	psock    = tools.JoinPathFromHere("extsvr.sock")
	chktimer = 60
	version  = "0.0.0"
)

var (
	ver = flag.Bool("version", false, "print version number and exit")
)

func init() {
	os.MkdirAll(tools.JoinPathFromHere("log"), 0775)
	flag.Parse()
	if *ver {
		println(version)
		os.Exit(0)
	}
	godaemon.Start()
}
func keepSvrRunning() {
	// 检查所有enable==true && manualStop==false的服务状态
	svrconf.xrange(func(key string, value *serviceParams) bool {
		if !value.Enable || value.manualStop {
			return true
		}
		if svrIsRunning(value) {
			return true
		}
		startSvr(key, value)
		time.Sleep(time.Millisecond * 500)
		return true
	})
}
func startSvr(name string, svr *serviceParams) string {
	defer func() { svrconf.manualstop(name, false) }()
	msg := ""
	dir := filepath.Dir(svr.Exec)
	params := []string{"--start", "-d", dir, "--background", "-m", "-p", "/tmp/" + name + ".pid", "--exec", svr.Exec, "--"} // "-d", dir,
	params = append(params, svr.Params...)
	cmd := exec.Command("start-stop-daemon", params...)
	b, err := cmd.CombinedOutput()
	if err != nil {
		msg = "=== start " + name + " error: " + err.Error() + " >> " + tools.String(b)
		stdlog.Error(msg)
		return msg
	}
	msg = "=== start " + name + " done. " + tools.String(b)
	stdlog.Info(msg)
	return msg
}

func stopSvr(name string, svr *serviceParams) string {
	defer func() { svrconf.manualstop(name, true) }()
	msg := ""
	params := []string{"--stop", "-p", "/tmp/" + name + ".pid"}
	cmd := exec.Command("start-stop-daemon", params...)
	b, err := cmd.CombinedOutput()
	if err != nil {
		msg = "*** stop " + name + " error: " + err.Error() + " >> " + tools.String(b)
		stdlog.Error(msg)
		return msg
	}
	msg = "*** stop " + name + " done. " + tools.String(b)
	stdlog.Warning(msg)
	return msg
}

func statusSvr(name string, svr *serviceParams) string {
	ss := strings.Builder{}
	ss.WriteString(fmt.Sprintf("Service:\t%s\n    Exec:\t%s\n    Params:\t%v\n    Enable:\t%v\n-------\n", name, svr.Exec, svr.Params, svr.Enable))
	ss.WriteString(psSvr(svr))
	return ss.String()
}
func psSvr(svr *serviceParams) string {
	s := []string{"-C", filepath.Base(svr.Exec), "-o", "user=", "-o", "pid=", "-o", `%cpu=`, "-o", `%mem=`, "-o", "stat=", "-o", "start=", "-o", "time=", "-o", "cmd="}
	cmd := exec.Command("ps", s...)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(tools.String(b))
}
func svrIsRunning(svr *serviceParams) bool {
	s := []string{"-C", filepath.Base(svr.Exec), "-o", "cmd="}
	cmd := exec.Command("ps", s...)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	out := tools.String(b)
	if len(svr.Params) == 0 {
		return strings.TrimSpace(out) == svr.Exec
	}
	found := true
	for _, v := range svr.Params {
		if !strings.Contains(out, v) {
			found = false
			break
		}
	}
	return found
}

type unixClient struct {
	conn  *net.UnixConn
	cache bytes.Buffer
	buf   []byte
}

func (uc *unixClient) Send(name, s string) {
	// s = fmt.Sprintf(sendfmt, name, s)
	if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	uc.conn.Write(tools.Bytes(s))
	stdlog.Info(">>> " + s)
}

// 接收消息格式： fmt.Sprintf("%d|%s|%s|%s|",do,name,exec,params)
// name: 服务名称
// do: 固定1字符 0-关闭链接，1-启动，2-停止，3-启用，4-停用，5-查询状态，6-删除服务配置，7-新增服务配置，8-初始化一个文件，9-列出所有配置，10-重启指定服务，98-刷新配置，99-停止
// exec: 要执行的文件完整路径（仅新增时有效）
// params：要执行的参数，多个参数用`，`分割，（仅新增时有效）
//
// 发送消息格式： fmt.Sprintf("%s",detail)
// detail: 消息内容
func main() {
	stdlog = logger.NewLogger(tools.JoinPathFromHere("log"), "extsvr", 10, 7)
	stdlog.System("start listen from unix socket")
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGSTOP)
	go func(c chan os.Signal) {
		sig := <-c // 监听关闭
		stdlog.System(fmt.Sprintf("caught signal %s: shutting down.", sig))
		os.Remove(psock)
		time.Sleep(time.Millisecond * 300)
		os.Exit(0)
	}(sigc)

	svrconf.readfile()
	loopfunc.LoopFunc(func(params ...interface{}) {
		uln, err := net.ListenUnix("unix", &net.UnixAddr{Name: psock, Net: "unix"})
		if err != nil {
			stdlog.Error("listen from unix socket error: " + err.Error())
			time.Sleep(time.Second)
			os.Exit(1)
		}
		uln.SetUnlinkOnClose(true)
		// 监听客户端
		go loopfunc.LoopFunc(func(params ...interface{}) {
			uln := params[0].(*net.UnixListener)
			for {
				// ln.SetDeadline(time.Now().Add(time.Second * 5))
				fd, err := uln.AcceptUnix()
				if err != nil {
					if strings.Contains(err.Error(), net.ErrClosed.Error()) {
						println("listener close")
						return
					}
					stdlog.Error("accept error: " + err.Error())
					continue
				}
				go recv(&unixClient{
					conn:  fd,
					buf:   make([]byte, 2048),
					cache: bytes.Buffer{},
				})
			}
		}, "unix accept", stdlog.DefaultWriter(), uln)

		// 循环检查进程，监听消息
		t := time.NewTicker(time.Second * time.Duration(chktimer))
		tc := time.NewTicker(time.Second * 10)
		for {
			select {
			case <-tc.C: // 检查socket文件
				if !tools.IsExist(psock) {
					uln.Close()
					// panic(fmt.Errorf("unix socket file is missing"))
					godaemon.RunBackground()
				}
			case <-t.C: // 定时检查服务
				keepSvrRunning()
			}
		}
	}, "main proc", stdlog.DefaultWriter())
}

func recv(cli *unixClient) {
	defer func() {
		if err := recover(); err != nil {
			stdlog.Error(err.(error).Error())
		}
		cli.conn.Close()
	}()
	for {
		cli.conn.SetReadDeadline(time.Now().Add(time.Minute))
		n, err := cli.conn.Read(cli.buf)
		if err != nil {
			if err != io.EOF {
				stdlog.Error("recv error: " + err.Error())
			}
			return
		}
		// 切割
		stdlog.Info("<<< " + tools.String(cli.buf[:n]))
		cli.cache.Write(cli.buf[:n])
		s := cli.cache.String()
		cli.cache.Reset()
		ss := strings.Split(s, "|")
		if len(ss) < 5 {
			cli.cache.WriteString(s)
			continue
		}
		if len(ss) > 5 {
			cli.cache.WriteString(strings.Join(ss[4:], "|"))
		}
		svrname := strings.TrimSpace(ss[1])
		svrdo := ss[0]
		v, ok := svrconf.load(svrname)
		switch svrdo {
		case "0": // 关闭连接
			return
		case "1": // 启动
			if !ok && svrname != "all" {
				cli.Send(svrname, "unknow server name: "+svrname)
				continue
			}
			if svrname == "all" {
				svrconf.xrange(func(key string, value *serviceParams) bool {
					if value.Enable {
						cli.Send(key, startSvr(key, value))
					}
					return true
				})
			} else {
				cli.Send(svrname, startSvr(svrname, v))
			}
		case "2": // 停止
			if !ok && svrname != "all" {
				cli.Send(svrname, "unknow server name: "+svrname)
				continue
			}
			if svrname == "all" {
				svrconf.xrange(func(key string, value *serviceParams) bool {
					if key == "ttyd" || key == "caddy" {
						return true
					}
					cli.Send(key, stopSvr(key, value))
					return true
				})
			} else {
				cli.Send(svrname, stopSvr(svrname, v))
			}
		case "3": // 启用
			if !ok {
				cli.Send(svrname, "unknow server name: "+svrname)
				continue
			}
			svrconf.setenable(svrname, true)
			svrconf.savefile()
			cli.Send(svrname, svrname+" set enable")
		case "4": // 停用
			if !ok {
				cli.Send(svrname, "unknow server name: "+svrname)
				continue
			}
			svrconf.setenable(svrname, false)
			svrconf.savefile()
			cli.Send(svrname, svrname+" set disable")
		case "5": // 状态
			if !ok {
				cli.Send(svrname, "unknow server name: "+svrname)
				continue
			}
			cli.Send(svrname, statusSvr(svrname, v))
		case "6": // 删除
			svrconf.delete(svrname)
			svrconf.savefile()
			cli.Send(svrname, svrname+" deleted")
		case "7": // 新增
			svrconf.store(svrname, ss[2], strings.Split(ss[3], ",")...)
			svrconf.savefile()
			cli.Send(svrname, svrname+" added")
		case "8": // 初始化一个文件
			svrconf.init()
			svrconf.savefile()
			cli.Send(svrname, "config file init done")
		case "9", "98": // list,update
			if svrdo == "98" {
				svrconf.readfile()
			}
			cmd := exec.Command("cat", svrconf.yamlfile.Fullpath())
			b, err := cmd.CombinedOutput()
			if err != nil {
				cli.Send("", err.Error())
				return
			}
			cli.Send("", tools.String(b))
		case "10": // 重启服务
			if !ok {
				cli.Send(svrname, "unknow server name: "+svrname)
				continue
			}
			cli.Send(svrname, stopSvr(svrname, v))
			cli.Send(svrname, startSvr(svrname, v))
		case "99":
			stdlog.System("client ask me to shut down")
			sigc <- syscall.SIGTERM
		}
	}
}
