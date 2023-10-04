// Package main 使用unix socket实现的一个控制台管理程序，利用start-stop-daemon实现进程管理
package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/xyzj/gopsu/config"
	"github.com/xyzj/gopsu/gocmd"
	"github.com/xyzj/gopsu/logger"
	"github.com/xyzj/gopsu/loopfunc"
	"github.com/xyzj/gopsu/pathtool"
	"gopkg.in/yaml.v3"
)

type serviceParams struct {
	Params     []string `yaml:"params"`
	Env        []string `yaml:"env"`
	Exec       string   `yaml:"exec"`
	Enable     bool     `yaml:"enable"`
	manualStop bool     `yaml:"-"`
}

var (
	stdlog   logger.Logger
	svrconf  *config.Formatted[serviceParams] // = mapPS{locker: sync.RWMutex{}, data: make(map[string]*serviceParams), yamlfile: yaml.New(pathtool.JoinPathFromHere("extsvr.yaml"))}
	sendfmt  = `%20s|%s|`
	sigc     = make(chan os.Signal, 1)
	psock    = pathtool.JoinPathFromHere("extsvr.sock")
	chktimer = 60
	version  = "0.0.0"
)

func initconfig() {
	svrconf.PutItem("caddy", &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/caddy",
		Params: []string{"run"},
	})
	svrconf.PutItem("ttyd", &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/ttyd",
		Params: []string{"-p 7681", "-m 3", "login"},
	})
	svrconf.PutItem("backend", &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/backend",
		Params: []string{"-portable", "-conf=backend.conf", "-http=6819", "-forcehttp=true"},
	})
	svrconf.PutItem("uas", &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/uas",
		Params: []string{"-portable", "-conf=luwak.conf", "-http=6820", "-forcehttp=false"},
	})
	svrconf.PutItem("ecms-mod", &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/ecms-mod",
		Params: []string{"-portable", "-conf=ecms.conf", "-http=6821", "-tcp=6828", "-tcpmodule=wlst", "-forcehttp=false"},
	})
	svrconf.PutItem("task", &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/task",
		Params: []string{"-portable", "-conf=luwak.conf", "-http=6822", "-forcehttp=false"},
	})
	svrconf.PutItem("event", &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/logger",
		Params: []string{"-portable", "-conf=luwak.conf", "-http=6823", "-forcehttp=false"},
	})
	svrconf.PutItem("adc", &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/adc",
		Params: []string{"-portable", "-conf=luwak.conf", "-http=6832", "-forcehttp=false"},
	})
	svrconf.PutItem("ccb", &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/ccb",
		Params: []string{"-portable", "-conf=luwak.conf", "-http=6836", "-forcehttp=false"},
	})
	svrconf.PutItem("alarm", &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/alarmlog",
		Params: []string{"-portable", "-conf=luwak.conf", "-http=6837", "-forcehttp=false"},
	})
	svrconf.PutItem("msgpush", &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/msgpush",
		Params: []string{"-portable", "-conf=luwak.conf", "-http=6824", "-forcehttp=false"},
	})
	svrconf.PutItem("bsjk", &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/businessjk",
		Params: []string{"-portable", "-conf=luwak.conf", "-http=6827", "-forcehttp=false"},
	})
	svrconf.PutItem("uiact", &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/netcore/userinteraction",
		Params: []string{"--log=/opt/bin/log/userinteraction", "--conf=/opt/bin/conf/userinteraction"},
	})
	svrconf.PutItem("dpwlst", &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/netcore/dataparser-wlst",
		Params: []string{"--log=/opt/bin/log/dataparser-wlst", "--conf=/opt/bin/conf/dataparser-wlst"},
	})
	svrconf.PutItem("dm", &serviceParams{
		Enable: true,
		Exec:   "/opt/bin/netcore/datamaintenance",
		Params: []string{"--log=/opt/bin/log/datamaintenance", "--conf=/opt/bin/conf/datamaintenance"},
	})
	// 默认不起动的
	svrconf.PutItem("asset", &serviceParams{
		Enable: false,
		Exec:   "/opt/bin/assetmanager",
		Params: []string{"-portable", "-conf=luwak.conf", "-http=6825", "-forcehttp=false"},
	})
	svrconf.PutItem("gis", &serviceParams{
		Enable: false,
		Exec:   "/opt/bin/gismanager",
		Params: []string{"-portable", "-conf=luwak.conf", "-http=6826", "-forcehttp=false"},
	})
	svrconf.PutItem("nboam", &serviceParams{
		Enable: false,
		Exec:   "/opt/bin/nboam",
		Params: []string{"-portable", "-conf=nboam.conf", "-http=6835", "-forcehttp=false"},
	})
	svrconf.PutItem("ftpupg", &serviceParams{
		Enable: false,
		Exec:   "/opt/bin/ftpupgrade",
		Params: []string{"-portable", "-conf=ftp.conf", "-http=6829", "-ftp=6830", "-forcehttp=false"},
	})
	svrconf.PutItem("dpnb", &serviceParams{
		Enable: false,
		Exec:   "/opt/bin/netcore/dataparser-nbiot",
		Params: []string{"--log=/opt/bin/log/dataparser-nbiot", "--conf=/opt/bin/conf/dataparser-nbiot"},
	})
}
func manualstop(name string, stop bool) {
	v, ok := svrconf.GetItem(name)
	if !ok {
		return
	}
	v.manualStop = stop
	svrconf.PutItem(name, v)
}
func setenable(name string, en bool) {
	v, ok := svrconf.GetItem(name)
	if !ok {
		return
	}
	v.Enable = en
	svrconf.PutItem(name, v)
}
func keepSvrRunning() {
	// 检查所有enable==true && manualStop==false的服务状态
	svrconf.ForEach(func(key string, value *serviceParams) bool {
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
	defer func() { manualstop(name, false) }()
	msg := ""
	dir := filepath.Dir(svr.Exec)
	params := []string{"--start", "-d", dir, "--background", "-m", "-p", "/tmp/" + name + ".pid", "--exec", svr.Exec, "--"} // "-d", dir,
	params = append(params, svr.Params...)
	cmd := exec.Command("start-stop-daemon", params...)
	if len(svr.Env) > 0 {
		cmd.Env = svr.Env
	}
	b, err := cmd.CombinedOutput()
	if err != nil {
		msg = "[START]:\n" + name + " error: " + err.Error() + " >> " + string(b)
		stdlog.Error(msg)
		return msg
	}
	pid := ""
	bb, err := os.ReadFile("/tmp/" + name + ".pid")
	if err == nil {
		pid = strings.TrimSpace(string(bb))
	}
	msg = "[START]:\n" + name + " done. PID: " + pid + "\n|>> " + svr.Exec + " " + strings.Join(svr.Params, " ")
	if len(b) > 0 {
		msg += "\n|>> " + string(b)
	}
	msg += "\n\n"
	stdlog.Info(msg)
	return msg
}

func stopSvr(name string, svr *serviceParams) string {
	defer func() { manualstop(name, true) }()
	pid := ""
	bb, err := os.ReadFile("/tmp/" + name + ".pid")
	if err == nil {
		pid = strings.TrimSpace(string(bb))
	}
	msg := ""
	params := []string{"--stop", "-p", "/tmp/" + name + ".pid"}
	cmd := exec.Command("start-stop-daemon", params...)
	b, err := cmd.CombinedOutput()
	if err != nil {
		msg = "[STOP]:\n" + name + " error: " + err.Error() + " >> " + string(b)
		stdlog.Error(msg)
		return msg
	}
	msg = "[STOP]:\n" + name + " done. PID: " + pid
	if len(b) > 0 {
		msg += "\n|>> " + string(b)
	}
	msg += "\n\n"
	stdlog.Warning(msg)
	return msg
}

func statusSvr(name string, svr *serviceParams) string {
	ss := strings.Builder{}
	b, err := yaml.Marshal(svr)
	if err == nil {
		ss.WriteString("[CONFIG]:\n")
		ss.Write(b)
		ss.WriteByte(10)
	}
	// ss.WriteString(fmt.Sprintf("Service:\t%s\n    Exec:\t%s\n    Params:\t%v\n    Env:\t%v\n    Enable:\t%v\nProcess:\n", name, svr.Exec, svr.Params, svr.Env, svr.Enable))
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
	return "[PS]:\n" + strings.TrimSpace(string(b)) + "\n\n"
}
func svrIsRunning(svr *serviceParams) bool {
	s := []string{"-C", filepath.Base(svr.Exec), "-o", "cmd="}
	cmd := exec.Command("ps", s...)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	out := string(b)
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
	uc.conn.Write([]byte(s))
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
	os.MkdirAll(pathtool.JoinPathFromHere("log"), 0775)
	// flag.Parse()
	// if *ver {
	// 	println(version)
	// 	os.Exit(0)
	// }

	gocmd.DefaultProgram(&gocmd.Info{
		Ver: version,
	}).OnSignalQuit(func() { os.Remove(psock) }).ExecuteDefault("start")
	// godaemon.Start(func() {
	// 	os.Remove(psock)
	// 	stdlog.System(fmt.Sprintf("got the signal, shutting down."))
	// })
	stdlog = logger.NewLogger(pathtool.JoinPathFromHere("log"), "extsvr", 10, 7, false)
	stdlog.System("start listen from unix socket")
	// signal.Notify(sigc, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	// go func(c chan os.Signal) {
	// 	sig := <-c // 监听关闭
	// 	stdlog.System(fmt.Sprintf("caught signal %s: shutting down.", sig))
	// 	os.Remove(psock)
	// 	time.Sleep(time.Millisecond * 300)
	// 	os.Exit(0)
	// }(sigc)
	svrconf = config.NewFormatFile[serviceParams](pathtool.JoinPathFromHere("extsvr.yaml"), config.YAML)
	// println(svrconf.Print())
	// svrconf.readfile()
	go loopfunc.LoopFunc(func(params ...interface{}) {
		tc := time.NewTicker(time.Minute)
		tc2 := time.NewTicker(time.Second * 13)
		for {
			select {
			case <-tc.C:
				keepSvrRunning()
			case <-tc2.C:
				if pathtool.IsExist(psock) {
					continue
				}
				cmd := exec.Command(os.Args[0], "restart")
				cmd.Start()
			}
		}
	}, "keeprunning", nil)

	loopfunc.LoopFunc(func(params ...interface{}) {
		uln, err := net.ListenUnix("unix", &net.UnixAddr{Name: psock, Net: "unix"})
		if err != nil {
			stdlog.Error("listen from unix socket error: " + err.Error())
			time.Sleep(time.Second)
			os.Exit(1)
		}
		uln.SetUnlinkOnClose(true)

		// 监听客户端
		for {
			// ln.SetDeadline(time.Now().Add(time.Second * 5))
			fd, err := uln.AcceptUnix()
			if err != nil {
				if strings.Contains(err.Error(), net.ErrClosed.Error()) {
					panic(fmt.Errorf("listener close"))
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
		stdlog.Info("<<< " + string(cli.buf[:n]))
		cli.cache.Write(cli.buf[:n])
	RECV:
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
		v, ok := svrconf.GetItem(svrname)
		switch svrdo {
		case "0": // 关闭连接
			return
		case "1": // 启动
			if !ok && svrname != "all" {
				cli.Send(svrname, "unknow server name: "+svrname)
				goto RECV
			}
			if svrname == "all" {
				svrconf.ForEach(func(key string, value *serviceParams) bool {
					if value.Enable {
						cli.Send(key, startSvr(key, value))
						time.Sleep(time.Second * 2)
						cli.Send(key, statusSvr(key, value))
					}
					return true
				})
			} else {
				cli.Send(svrname, startSvr(svrname, v))
				time.Sleep(time.Second * 2)
				cli.Send(svrname, statusSvr(svrname, v))
			}
		case "2": // 停止
			if !ok && svrname != "all" {
				cli.Send(svrname, "unknow server name: "+svrname)
				goto RECV
			}
			if svrname == "all" {
				svrconf.ForEach(func(key string, value *serviceParams) bool {
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
				goto RECV
			}
			setenable(svrname, true)
			svrconf.ToFile()
			cli.Send(svrname, "=== "+svrname+" set enable")
		case "4": // 停用
			if !ok {
				cli.Send(svrname, "unknow server name: "+svrname)
				goto RECV
			}
			setenable(svrname, false)
			svrconf.ToFile()
			cli.Send(svrname, "*** "+svrname+" set disable")
		case "5": // 状态
			if !ok {
				cli.Send(svrname, "unknow server name: "+svrname)
				goto RECV
			}
			cli.Send(svrname, statusSvr(svrname, v))
		case "6": // 删除
			svrconf.DelItem(svrname)
			svrconf.ToFile()
			cli.Send(svrname, "--- "+svrname+" removed")
		case "7": // 新增
			svrconf.PutItem(svrname, &serviceParams{Exec: ss[2], Params: strings.Split(ss[3], " "), Enable: true})
			// svrconf.PutItem(svrname, ss[2], strings.Split(ss[3], ",")...)
			svrconf.ToFile()
			cli.Send(svrname, "+++ "+svrname+" added")
		case "8": // 初始化一个文件
			initconfig()
			svrconf.ToFile()
			cli.Send(svrname, "config file init done")
		case "9", "98": // list,update
			if svrdo == "98" {
				svrconf.FromFile("")
			}
			cli.Send("", svrconf.Print())
		case "10": // 重启服务
			if !ok {
				cli.Send(svrname, "unknow server name: "+svrname)
				goto RECV
			}
			cli.Send(svrname, stopSvr(svrname, v))
			time.Sleep(time.Second * 2)
			cli.Send(svrname, startSvr(svrname, v))
			time.Sleep(time.Second * 2)
			cli.Send(svrname, statusSvr(svrname, v))
		case "99":
			stdlog.System("client ask me to shut down")
			sigc <- syscall.SIGTERM
		}
		if len(ss[4:]) >= 5 {
			goto RECV
		}
	}
}
