package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	model "extsvr/model"

	"github.com/xyzj/gopsu/config"
	"github.com/xyzj/gopsu/gocmd"
	"github.com/xyzj/gopsu/logger"
	"github.com/xyzj/gopsu/loopfunc"
	"github.com/xyzj/gopsu/pathtool"
	"gopkg.in/yaml.v3"
)

var systemd = `# 使用说明:
# 1. copy %s.service to /etc/systemd/system
# 2. sudo systemctl daemon-reload && sudo systemctl start ssdctld && sudo systemctl enable ssdctld

[Unit]
Description=services manager daemon
After=network.target

[Service]
EnvironmentFile=
User=%s
Group=%s
WorkingDirectory=%s
ExecStart=%s run
ExecReload=%s restart
ExecStop=%s stop
ExecStopPost=/usr/bin/rm -f %s
Type=simple
KillMode=none
Restart=on-failure
RestartSec=42s

[Install]
WantedBy=multi-user.target`

var (
	stdlog     logger.Logger
	svrconf    *config.Formatted[model.ServiceParams]
	exename    = pathtool.GetExecName()
	psock      = pathtool.JoinPathFromHere("ssdctld.sock")
	confile    = pathtool.JoinPathFromHere("ssdctld.yaml")
	confileOld = pathtool.JoinPathFromHere("extsvr.yaml")
	logdir     = pathtool.JoinPathFromHere("log")

	app     *gocmd.Program
	version = "0.0.0"
)

type unixClient struct {
	conn *net.UnixConn
	buf  []byte
}

func (uc *unixClient) Send(name, s string) {
	if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	uc.conn.Write([]byte(s))
	stdlog.Info(">>> " + s)
}

func main() {
	if pathtool.IsExist(confileOld) {
		os.Rename(confileOld, confile)
	}
	os.MkdirAll(logdir, 0o775)
	app = gocmd.DefaultProgram(&gocmd.Info{
		Title: "programs managerment",
		Ver:   version,
		Descript: `use "start-stop-daemon" to manager process

ssdctld.yaml.sample:
app1:                    // program name
  enable: true           // enable autostart and timer check
  exec: /op/aa           // program exec path
  dir: /op               // program working dir, default is program's base dir
  params:                // program args
    - -q=12
    - -c=$pubip          // '$public' will be replaced by the replace setting before run
  env:                   // set the sys env, should be 'key=value' format
    - https_proxy=http:127.0.0.1:8080
  replace:               // params replacer, can replace params variable before run, should be 'key=value' format, and key must start with '$'
    - $pubip=curl -s 4.ipw.cn
  log2file: true         // save program stdout to /tmp/[program name].log

in this case, $pubip will be replace to the result of 'curl -s 4.ipw.cn'`,
	}).
		AddCommand(&gocmd.Command{
			Name:     "systemd",
			Descript: "create a systemd service file",
			RunWithExitCode: func(pi *gocmd.ProcInfo) int {
				var uname, ugrp string
				u, err := user.Current()
				if err == nil {
					uname = u.Uid
					ugrp = u.Gid
				}
				g, err := user.LookupGroupId(u.Gid)
				if err == nil {
					uname = u.Username
					ugrp = g.Name
				}
				os.WriteFile(pathtool.JoinPathFromHere(exename+".service"), []byte(fmt.Sprintf(systemd,
					uname, ugrp,
					exename,
					pathtool.GetExecDir(),
					pathtool.GetExecFullpath(),
					pathtool.GetExecFullpath(),
					pathtool.GetExecFullpath(),
					psock,
				)), 0o664)
				println(fmt.Sprintf(`create systemd service file done.
1. copy "%s.service" to "/etc/systemd/system/"
2. run "systemctl daemon-reload" as root
3. run "systemctl start %s && system enable %s" as root`, exename, exename, exename))
				return 0
			},
		}).
		AfterStop(func() {
			os.Remove(psock)
			time.Sleep(time.Millisecond * 300)
		}).
		BeforeStart(func() {
			if pathtool.IsExist(psock) {
				os.Remove(psock)
			}
		})
	app.ExecuteDefault("start")
	// 初始化
	if os.Getenv("ssdctld_stdlog") == "console" {
		stdlog = logger.NewConsoleLogger()
	} else {
		stdlog = logger.NewLogger(logdir, exename, 10, 7, false)
	}
	stdlog.System("start listen from unix socket")
	svrconf = config.NewFormatFile[model.ServiceParams](confile, config.YAML)
	chanRecv := make(chan *unixClient, 10)
	// 后台处理
	go loopfunc.LoopFunc(func(params ...interface{}) {
		tKeep := time.NewTicker(time.Minute)
		for {
			select {
			case <-tKeep.C:
				// 检查所有enable==true && manualStop==false的服务状态
				svrconf.ForEach(func(key string, value *model.ServiceParams) bool {
					if !value.Enable || value.ManualStop {
						return true
					}
					if svrIsRunning(value) {
						return true
					}
					startSvr(key, value)
					time.Sleep(time.Millisecond * 500)
					return true
				})
			case cli := <-chanRecv:
				recv(cli)
			}
		}
	}, "recv", stdlog.DefaultWriter())
	// 开始监听
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
			fd, err := uln.AcceptUnix()
			if err != nil {
				if strings.Contains(err.Error(), net.ErrClosed.Error()) {
					panic(fmt.Errorf("listener close"))
				}
				stdlog.Error("accept error: " + err.Error())
				continue
			}
			chanRecv <- &unixClient{
				conn: fd,
				buf:  make([]byte, 2048),
				// cache: bytes.Buffer{},
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
		stdlog.Info("<<< " + string(cli.buf[:n]))
		for _, v := range bytes.Split(cli.buf[:n], []byte{10}) {
			if len(v) == 0 {
				continue
			}
			todo := &model.ToDo{}
			todo.FromJSON(v)
			exe, ok := svrconf.GetItem(todo.Name)
			switch todo.Do {
			case model.JobShutdown:
				stdlog.System("client ask me to shut down")
				app.Exit(0)
			case model.JobClose: // 关闭
				return
			case model.JobStart: // 启动
				if !ok && todo.Name != "all" {
					cli.Send(todo.Name, "unknow server name: "+todo.Name)
					continue
				}
				if todo.Name == "all" {
					svrconf.ForEach(func(key string, value *model.ServiceParams) bool {
						if !value.Enable {
							return true
						}
						cli.Send(key, startSvr(key, value))
						time.Sleep(time.Second * 2)
						cli.Send(key, statusSvr(key, value))
						return true
					})
				} else {
					cli.Send(todo.Name, startSvr(todo.Name, exe))
					time.Sleep(time.Second * 2)
					cli.Send(todo.Name, statusSvr(todo.Name, exe))
				}
			case model.JobStop: // 停止
				if !ok && todo.Name != "all" {
					cli.Send(todo.Name, "unknow server name: "+todo.Name)
					continue
				}
				if todo.Name == "all" {
					svrconf.ForEach(func(key string, value *model.ServiceParams) bool {
						if key == "ttyd" || key == "caddy" {
							return true
						}
						cli.Send(key, stopSvr(key, value))
						return true
					})
				} else {
					cli.Send(todo.Name, stopSvr(todo.Name, exe))
				}
			case model.JobRestart: // 重启
				if !ok && todo.Name != "all" {
					cli.Send(todo.Name, "unknow server name: "+todo.Name)
					continue
				}
				if todo.Name == "all" {
					svrconf.ForEach(func(key string, value *model.ServiceParams) bool {
						if key == "ttyd" || key == "caddy" {
							return true
						}
						s := stopSvr(todo.Name, exe)
						cli.Send(todo.Name, s)
						if !strings.Contains(s, "No such process") {
							i := 7
							for i > 0 {
								i--
								time.Sleep(time.Millisecond * 500)
								if !svrIsRunning(value) {
									break
								}
							}
						}
						cli.Send(todo.Name, startSvr(todo.Name, exe))
						time.Sleep(time.Second * 2)
						cli.Send(todo.Name, statusSvr(todo.Name, exe))
						return true
					})
				} else {
					s := stopSvr(todo.Name, exe)
					cli.Send(todo.Name, s)
					if !strings.Contains(s, "No such process") {
						i := 7
						for i > 0 {
							i--
							time.Sleep(time.Millisecond * 500)
							if !svrIsRunning(exe) {
								break
							}
						}
					}
					cli.Send(todo.Name, startSvr(todo.Name, exe))
					time.Sleep(time.Second * 2)
					cli.Send(todo.Name, statusSvr(todo.Name, exe))
				}
			case model.JobStatus: // 状态查询
				if !ok && todo.Name != "all" {
					cli.Send(todo.Name, "unknow server name: "+todo.Name)
					continue
				}
				if todo.Name == "all" {
					svrconf.ForEach(func(key string, value *model.ServiceParams) bool {
						if value.Enable {
							cli.Send(key, statusSvr(key, value))
						}
						return true
					})
				} else {
					cli.Send(todo.Name, statusSvr(todo.Name, exe))
				}
			case model.JobEnable: // 启用
				if !ok {
					cli.Send(todo.Name, "unknow server name: "+todo.Name)
					continue
				}
				exe.Enable = true
				svrconf.PutItem(todo.Name, exe)
				svrconf.ToFile()
				cli.Send(todo.Name, ">>> "+todo.Name+" enabled")
			case model.JobDisable: // 停用
				if !ok {
					cli.Send(todo.Name, "unknow server name: "+todo.Name)
					continue
				}
				exe.Enable = false
				svrconf.PutItem(todo.Name, exe)
				svrconf.ToFile()
				cli.Send(todo.Name, ">>> "+todo.Name+" disabled")
			case model.JobRemove: // 删除服务
				svrconf.DelItem(todo.Name)
				svrconf.ToFile()
				cli.Send(todo.Name, "--- "+todo.Name+" removed")
			case model.JobCreate: // 新增服务
				if todo.Name == "all" {
					cli.Send("all", "can not use 'all' as application's name")
					return
				}
				svrconf.PutItem(todo.Name, &model.ServiceParams{
					Exec:   todo.Exec,
					Params: todo.Params,
					Enable: true,
				})
				svrconf.ToFile()
				cli.Send(todo.Name, "+++ "+todo.Name+" added")
			case model.JobList, model.JobUpate: // 列出所有，刷新
				if todo.Do == model.JobUpate {
					svrconf.FromFile(confile)
				}
				cli.Send("", svrconf.Print())
			}
		}
	}
}

func startSvr(name string, svr *model.ServiceParams) string {
	defer func() { manualstop(name, false) }()
	// 准备替换内容
	parmrepl := strings.NewReplacer()
	if len(svr.Replace) > 0 {
		xss := []string{}
		for _, v := range svr.Replace {
			if !strings.HasPrefix(v, "$") {
				continue
			}
			ss := strings.Split(v, "=")
			if len(ss) != 2 {
				continue
			}
			yss := strings.Split(ss[1], " ")
			cmd := exec.Command(yss[0], yss[1:]...)
			b, err := cmd.CombinedOutput()
			if err == nil {
				xss = append(xss, ss[0], strings.TrimSpace(string(b)))
			}
		}
		if len(xss) > 0 {
			parmrepl = strings.NewReplacer(xss...)
		}
	}
	// 设置目录
	if svr.Dir == "" {
		svr.Dir = filepath.Dir(svr.Exec)
	}
	params := []string{"--start", "--chdir=" + svr.Dir, "--background", "-m", "--remove-pidfile", "--pidfile=/tmp/" + name + ".pid"} //, "--output=/tmp/" + name + ".log", "--exec=" + svr.Exec} // "--background"
	if svr.Log2file {
		params = append(params, "--output=/tmp/"+name+".log")
	}
	params = append(params, "--exec="+svr.Exec)
	if len(svr.Params) > 0 {
		params = append(params, "--")
		if len(svr.Replace) == 0 {
			params = append(params, svr.Params...)
		} else {
			for _, v := range svr.Params {
				if strings.Contains(v, "$") {
					params = append(params, parmrepl.Replace(v))
				} else {
					params = append(params, v)
				}
			}
		}
	}
	cmd := exec.Command("start-stop-daemon", params...)
	if len(svr.Env) > 0 {
		cmd.Env = svr.Env
	}
	msg := ""
	b, err := cmd.CombinedOutput()
	if err != nil {
		msg = "[START] " + name + ":\nerror: " + err.Error() + " >> " + string(b)
		stdlog.Error(msg)
		return msg
	}
	pid := ""
	bb, err := os.ReadFile("/tmp/" + name + ".pid")
	if err == nil {
		pid = strings.TrimSpace(string(bb))
	}
	msg = "[START] " + name + ":\ndone. PID: " + pid + "\n|>> " + svr.Exec + " " + strings.Join(svr.Params, " ")
	if len(b) > 0 {
		msg += "\n|>> " + string(b)
	}
	msg += "\n"
	stdlog.Info(msg)
	return msg
}

func stopSvr(name string, _ *model.ServiceParams) string {
	defer func() { manualstop(name, true) }()
	pid := ""
	bb, err := os.ReadFile("/tmp/" + name + ".pid")
	if err == nil {
		pid = strings.TrimSpace(string(bb))
	}
	println("-------------", pid)
	msg := ""
	params := []string{"--stop", "--remove-pidfile", "-p", "/tmp/" + name + ".pid"}
	cmd := exec.Command("start-stop-daemon", params...)
	b, err := cmd.CombinedOutput()
	if err != nil {
		msg = "[STOP] " + name + ":\nerror: " + err.Error() + " >> " + string(b)
		stdlog.Error(msg)
		return msg
	}
	msg = "[STOP] " + name + ":\ndone. PID: " + pid
	if len(b) > 0 {
		msg += "\n|>> " + string(b)
	}
	msg += "\n"
	stdlog.Warning(msg)
	return msg
}

func statusSvr(name string, svr *model.ServiceParams) string {
	ss := strings.Builder{}
	b, err := yaml.Marshal(svr)
	if err == nil {
		ss.WriteString("[CONFIG] " + name + ":\n")
		ss.Write(b)
		ss.WriteByte(10)
	}
	// ss.WriteString(fmt.Sprintf("Service:\t%s\n    Exec:\t%s\n    Params:\t%v\n    Env:\t%v\n    Enable:\t%v\nProcess:\n", name, svr.Exec, svr.Params, svr.Env, svr.Enable))
	ss.WriteString(psSvr(svr))
	return ss.String()
}

func psSvr(svr *model.ServiceParams) string {
	s := []string{"-C", filepath.Base(svr.Exec), "-o", "user=", "-o", "pid=", "-o", `%cpu=`, "-o", `%mem=`, "-o", "stat=", "-o", "start=", "-o", "time=", "-o", "cmd="}
	cmd := exec.Command("ps", s...)
	b, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return "[PS]:\n" + strings.TrimSpace(string(b)) + "\n"
}

func svrIsRunning(svr *model.ServiceParams) bool {
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
		if strings.Contains(v, "$") {
			continue
		}
		if !strings.Contains(out, v) {
			found = false
			break
		}
	}
	return found
}

func manualstop(name string, stop bool) {
	v, ok := svrconf.GetItem(name)
	if !ok {
		return
	}
	v.ManualStop = stop
	svrconf.PutItem(name, v)
}
