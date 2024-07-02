package main

import (
	"bytes"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	model "extsvr/model"

	"github.com/xyzj/gopsu/gocmd"
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
	nokeepalive = flag.Bool("noka", false, "do not check and keep programs alive")
	// stdlog     logger.Logger
	exename    = pathtool.GetExecName()
	psock      = pathtool.JoinPathFromHere("ssdctld.sock")
	confile    = pathtool.JoinPathFromHere("ssdctld.yaml")
	confileOld = pathtool.JoinPathFromHere("extsvr.yaml")
	logdir     = pathtool.JoinPathFromHere("log")
	piddir     = pathtool.JoinPathFromHere("pid.d")
	cnfdir     = pathtool.JoinPathFromHere("cnf.d")
	allconf    *model.Config

	app     *gocmd.Program
	version = "0.0.0"
	// ps -C name
	// ps -p pid
	// psout   = []string{"-o", "pid=", "-o", "user=", "-o", `%cpu=`, "-o", `%mem=`, "-o", "stat=", "-o", "start=", "-o", "time=", "-o", "cmd="}
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
	// stdlog.Info(">>> " + s)
}

func main() {
	if !pathtool.IsExist(cnfdir) {
		os.Mkdir(cnfdir, 0o775)
	}
	if !pathtool.IsExist(piddir) {
		os.Mkdir(piddir, 0o775)
	}
	if !pathtool.IsExist(logdir) {
		os.Mkdir(logdir, 0o775)
	}
	if pathtool.IsExist(confileOld) {
		os.Rename(confileOld, confile)
	}
	allconf = model.NewCnf(cnfdir)

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
					exename,
					uname, ugrp,
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
		AddCommand(&gocmd.Command{
			Name:     "converold",
			Descript: "conver old config file to cnf.d",
			RunWithExitCode: func(pi *gocmd.ProcInfo) int {
				if allconf == nil {
					allconf = model.NewCnf(cnfdir)
					allconf.ConverFromOld()
				}
				return 0
			},
		}).
		AfterStop(func() {
			os.Remove(psock)
		}).
		BeforeStart(func() {
			time.Sleep(time.Millisecond * 500)
			if pathtool.IsExist(psock) {
				os.Remove(psock)
			}
		})
	app.ExecuteDefault("start")

	allconf.ConverFromOld()
	allconf.FromFiles()
	// stdlog.System("start listen from unix socket")
	chanRecv := make(chan *unixClient, 10)
	// 后台处理
	go loopfunc.LoopFunc(func(params ...interface{}) {
		tKeep := time.NewTicker(time.Minute)
		if *nokeepalive {
			tKeep.Stop()
		}
		for {
			select {
			case <-tKeep.C:
				// 检查所有enable==true && manualStop==false的服务状态
				allconf.ForEach(func(key string, value *model.ServiceParams) bool {
					if !value.Enable || value.ManualStop {
						return true
					}
					if _, _, ok := svrIsRunning(value); ok {
						return true
					}
					startSvrFork(key, value)
					time.Sleep(time.Millisecond * 500)
					return true
				})
			case cli := <-chanRecv:
				recv(cli)
			}
		}
	}, "recv", nil) // stdlog.DefaultWriter())
	// 开始监听
	loopfunc.LoopFunc(func(params ...interface{}) {
		uln, err := net.ListenUnix("unix", &net.UnixAddr{Name: psock, Net: "unix"})
		if err != nil {
			// stdlog.Error("listen from unix socket error: " + err.Error())
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
				// stdlog.Error("accept error: " + err.Error())
				continue
			}
			chanRecv <- &unixClient{
				conn: fd,
				buf:  make([]byte, 2048),
				// cache: bytes.Buffer{},
			}
		}
	}, "main proc", nil) // stdlog.DefaultWriter())
}

func recv(cli *unixClient) {
	defer func() {
		if err := recover(); err != nil {
			println("recv recover:" + err.(error).Error())
			// stdlog.Error(err.(error).Error())
		}
		cli.conn.Close()
	}()
	for {
		cli.conn.SetReadDeadline(time.Now().Add(time.Minute))
		n, err := cli.conn.Read(cli.buf)
		if err != nil {
			// if err != io.EOF {
			// println("recv error: " + err.Error())
			// stdlog.Error("recv error: " + err.Error())
			// }
			return
		}
		// 切割
		// stdlog.Info("<<< " + string(cli.buf[:n]))
		for _, v := range bytes.Split(cli.buf[:n], []byte{10}) {
			if len(v) == 0 {
				continue
			}
			todo := &model.ToDo{}
			todo.FromJSON(v)
			exe, ok := allconf.GetItem(todo.Name)
			switch todo.Do {
			case model.JobShutdown:
				// stdlog.System("client ask me to shut down")
				app.Exit(0)
			case model.JobClose: // 关闭
				return
			case model.JobStart: // 启动
				if !ok && todo.Name != "all" {
					cli.Send(todo.Name, "unknow server name: "+todo.Name)
					continue
				}
				if todo.Name == "all" {
					allconf.ForEach(func(key string, value *model.ServiceParams) bool {
						if !value.Enable {
							return true
						}
						s, ok := startSvrFork(key, value)
						cli.Send(key, s)
						if ok {
							// time.Sleep(time.Second * 1)
							cli.Send(key, statusSvr(key, value))
						}
						return true
					})
				} else {
					s, ok := startSvrFork(todo.Name, exe)
					cli.Send(todo.Name, s)
					if ok {
						// time.Sleep(time.Second * 1)
						cli.Send(todo.Name, statusSvr(todo.Name, exe))
					}
				}
			case model.JobStop: // 停止
				if !ok && todo.Name != "all" {
					cli.Send(todo.Name, "unknow server name: "+todo.Name)
					continue
				}
				if todo.Name == "all" {
					allconf.ForEach(func(key string, value *model.ServiceParams) bool {
						if !value.Enable {
							return true
						}
						if strings.Contains(value.Exec, "ttyd") ||
							strings.Contains(value.Exec, "caddy") ||
							strings.Contains(value.Exec, "dragonfly") ||
							strings.Contains(value.Exec, "stmq") {
							return true
						}
						cli.Send(key, stopSvrFork(key, value))
						return true
					})
				} else {
					cli.Send(todo.Name, stopSvrFork(todo.Name, exe))
				}
			case model.JobRestart: // 重启
				if !ok && todo.Name != "all" {
					cli.Send(todo.Name, "unknow server name: "+todo.Name)
					continue
				}
				if todo.Name == "all" {
					allconf.ForEach(func(key string, value *model.ServiceParams) bool {
						if !value.Enable {
							return true
						}
						if strings.Contains(value.Exec, "ttyd") ||
							strings.Contains(value.Exec, "caddy") ||
							strings.Contains(value.Exec, "dragonfly") ||
							strings.Contains(value.Exec, "stmq") {
							return true
						}
						cli.Send(key, stopSvrFork(key, value))
						time.Sleep(time.Second)
						// i := 7
						// for i > 0 {
						// 	i--
						// 	if _, _, ok := svrIsRunning(value); !ok {
						// 		break
						// 	}
						// 	time.Sleep(time.Millisecond * 500)
						// }
						s, ok := startSvrFork(key, value)
						cli.Send(key, s)
						if ok {
							// time.Sleep(time.Second * 1)
							cli.Send(key, statusSvr(key, value))
						}
						return true
					})
				} else {
					cli.Send(todo.Name, stopSvrFork(todo.Name, exe))
					time.Sleep(time.Second)
					// i := 7
					// for i > 0 {
					// 	i--
					// 	if _, _, ok := svrIsRunning(exe); !ok {
					// 		break
					// 	}
					// 	time.Sleep(time.Millisecond * 500)
					// }
					s, ok := startSvrFork(todo.Name, exe)
					cli.Send(todo.Name, s)
					if ok {
						// time.Sleep(time.Second * 1)
						cli.Send(todo.Name, statusSvr(todo.Name, exe))
					}
				}
			case model.JobStatus: // 状态查询
				if !ok && todo.Name != "all" {
					cli.Send(todo.Name, "unknow server name: "+todo.Name)
					continue
				}
				if todo.Name == "all" {
					allconf.ForEach(func(key string, value *model.ServiceParams) bool {
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
				allconf.SetEnable(todo.Name, true)
				cli.Send(todo.Name, ">>> "+todo.Name+" enabled")
			case model.JobDisable: // 停用
				if !ok {
					cli.Send(todo.Name, "unknow server name: "+todo.Name)
					continue
				}
				allconf.SetEnable(todo.Name, false)
				cli.Send(todo.Name, ">>> "+todo.Name+" disabled")
			case model.JobRemove: // 删除服务
				err := allconf.DelItem(todo.Name)
				if err != nil {
					cli.Send(todo.Name, "--- "+todo.Name+" remove failed: "+err.Error())
				} else {
					cli.Send(todo.Name, "--- "+todo.Name+" removed")
				}
			case model.JobCreate: // 新增服务
				if todo.Name == "all" {
					cli.Send("all", "can not use 'all' as application's name")
					return
				}
				err := allconf.AddItem(todo.Name, &model.ServiceParams{
					Exec:   todo.Exec,
					Params: todo.Params,
					Enable: true,
				})
				if err != nil {
					cli.Send(todo.Name, "+++ "+todo.Name+" add failed: "+err.Error())
				} else {
					cli.Send(todo.Name, "+++ "+todo.Name+" added")
				}
			case model.JobList, model.JobUpate: // 列出所有，刷新
				if todo.Do == model.JobUpate {
					allconf.FromFiles()
				}
				cli.Send("", allconf.Print())
			}
		}
	}
}

func statusSvr(name string, svr *model.ServiceParams) string {
	ss := strings.Builder{}
	_, ps, ok := svrIsRunning(svr)
	if !ok {
		ss.WriteString("[PS] " + name + " is not running\n\n")
		b, err := yaml.Marshal(svr)
		if err == nil {
			ss.WriteString("[CONFIG] " + name + ":\n")
			ss.Write(b)
			ss.WriteByte(10)
		}
	} else {
		ss.WriteString("[PS] " + name + ":\n" + ps)
	}
	return ss.String()
}

func svrIsRunning(svr *model.ServiceParams) (int, string, bool) {
	if svr.Pid > 0 { // 直接查/proc信息
		b, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", svr.Pid))
		if err == nil {
			return svr.Pid, fmt.Sprintf("%d\t%s", svr.Pid, bytes.ReplaceAll(b, []byte{0}, []byte{32})), true
		}
	}
	// 遍历/proc目录寻找
	pi := gocmd.QueryProcess(filepath.Base(svr.Exec))
	found := true
	for _, p := range pi {
		found = true
		for _, parm := range svr.Params {
			if strings.HasPrefix(parm, "$") {
				continue
			}
			if !strings.Contains(p.CmdLine, parm) {
				found = false
				break
			}
		}
		if found {
			return p.Pid, fmt.Sprintf("%d\t%s", p.Pid, p.CmdLine), true
		}
	}
	return 0, "", false
}

// func loadPid(name string) (int, bool) {
// 	b, err := os.ReadFile(filepath.Join(piddir, name+".pid"))
// 	if err != nil {
// 		return 0, false
// 	}
// 	pid, _ := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 32)
// 	if !gocmd.ProcessExist(int(pid)) {
// 		return 0, false
// 	}
// 	return int(pid), true
// }

func startSvrFork(name string, svr *model.ServiceParams) (string, bool) {
	if pid, ps, ok := svrIsRunning(svr); ok {
		if svr.Pid == 0 {
			svr.Pid = pid
		}
		return "[START] " + name + " is still running\n\n[PS] " + name + ":\n" + ps, false
	}
	var pid int
	var err error
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
	params := []string{svr.Exec}
	if len(svr.Params) > 0 {
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
	var msg string
	var f *os.File
	if svr.Log2file {
		f, _ = os.OpenFile(filepath.Join(logdir, name+".log"), os.O_CREATE|os.O_WRONLY, 0o664)
	}
	pa := &syscall.ProcAttr{
		Dir: svr.Dir,
		Env: os.Environ(),
		Sys: &syscall.SysProcAttr{
			Setpgid: true,
			// Setsid: true,
		},
	}
	if f != nil {
		pa.Files = []uintptr{0, f.Fd(), f.Fd()}
	}
	pid, err = syscall.ForkExec(svr.Exec, params, pa)
	if err != nil {
		msg = "[START] " + name + " error: " + err.Error()
		// stdlog.Error(msg)
		return msg, false
	}
	time.Sleep(time.Millisecond * 200)
	go func(pid int) {
		syscall.Wait4(pid, nil, 0, nil)
		// println("-----------pid out", pid)
	}(pid)
	if !gocmd.ProcessExist(pid) {
		return "[START] " + name + " failed", false
	}
	svr.ManualStop = false
	svr.Pid = pid
	os.WriteFile(filepath.Join(piddir, name+".pid"), []byte(fmt.Sprintf("%d", pid)), 0o664)
	msg = "[START] " + name + " done, PID: " + fmt.Sprintf("%d", pid) + "\n|>> " + svr.Exec + " " + strings.Join(svr.Params, " ")
	msg += "\n"
	// stdlog.Info(msg)
	return msg, true
}

func stopSvrFork(name string, svr *model.ServiceParams) string {
	pid, _, ok := svrIsRunning(svr)
	if !ok {
		return "[STOP] " + name + " is not running\n"
	}

	var msg string
	// println("---- before kill")
	err := syscall.Kill(pid, syscall.SIGTERM)
	if err != nil {
		msg = "[STOP] " + name + " error: " + err.Error()
		// stdlog.Error(msg)
		return msg
	}
	// println("---- before wait4")
	if svr.Pid == 0 {
		go func(pid int) {
			syscall.Wait4(pid, nil, syscall.WNOHANG, nil)
			// println("-----------stop pid out", pid)
		}(pid)
	}
	for i := 0; i < 7; i++ {
		time.Sleep(time.Millisecond * 500)
		// println("---- wait", gocmd.ProcessExist(pid))
		if !gocmd.ProcessExist(pid) {
			goto GOON
		}
	}
	syscall.Kill(pid, syscall.SIGKILL)
GOON:
	svr.ManualStop = true
	svr.Pid = 0
	os.Remove(filepath.Join(piddir, name+".pid"))
	time.Sleep(time.Millisecond * 200)
	msg = "[STOP] " + name + " done, PID: " + fmt.Sprintf("%d", pid)
	msg += "\n"
	// stdlog.Warning(msg)
	return msg
}
