package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	model "extsvr/model"

	gocmd "github.com/xyzj/go-cmd"
	"github.com/xyzj/toolbox"
	"github.com/xyzj/toolbox/logger"
	"github.com/xyzj/toolbox/loopfunc"
	"gopkg.in/yaml.v3"
)

var systemd = `# 使用说明:
# 1. copy %s.service to /etc/systemd/system
# 2. sudo systemctl daemon-reload && sudo systemctl start ssdctld && sudo systemctl enable ssdctld

[Unit]
Description=services manager daemon
After=network.target

[Service]
Environment="SSDCTLD_CHECK_SECONDS=60"
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
	nologger    = flag.Bool("nolog", false, "do not save control log")
	nokeepalive = flag.Bool("stopka", false, "do not check and keep programs alive")

	stdlog     logger.Logger
	exename    = gocmd.GetExecName()
	psock      = gocmd.JoinPathFromHere("ssdctld.sock")
	confile    = gocmd.JoinPathFromHere("ssdctld.yaml")
	confileOld = gocmd.JoinPathFromHere("extsvr.yaml")
	logdir     = gocmd.JoinPathFromHere("log")
	piddir     = gocmd.JoinPathFromHere("pid.d")
	cnfdir     = gocmd.JoinPathFromHere("cnf.d")
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
	b := strings.Builder{}
	bb := strings.Split(s, "\n")
	for k, v := range bb {
		if strings.HasPrefix(v, "[") ||
			strings.HasPrefix(v, "<") ||
			strings.HasPrefix(v, ">") ||
			strings.HasPrefix(v, "+") ||
			strings.HasPrefix(v, "-") ||
			strings.HasPrefix(v, "*") {
			b.WriteString(v)
		} else {
			b.WriteString("    " + v)
		}
		if k < len(bb)-1 {
			b.WriteByte(10)
		}
	}
	b.WriteByte(0)
	// if !strings.HasSuffix(s, "\x00") {
	// 	s += "\x00"
	// }
	uc.conn.Write([]byte(b.String()))
}

func main() {
	if !*nologger {
		stdlog = logger.NewLogger(logger.LogInfo,
			logger.OptFileDays(10),
			logger.OptFileDir(logdir),
			logger.OptFilename("ssdctld"))
	} else {
		stdlog = logger.NewNilLogger()
	}
	app = gocmd.DefaultProgram(&gocmd.Info{
		Title: "programs managerment",
		Ver:   version,
		Descript: `
ssdctld.yaml.sample:
app1:                    // program name
  priority: 999			 // start priority, from small to large
  startsec:				 // # of secs prog must stay up to be running
  exec: /op/aa           // program exec path
  dir: /op               // program working dir, default is program's base dir
  params:                // program args
    - -q=12
    - -c=$pubip          // '$public' will be replaced by the replace setting before run
  env:                   // set the sys env, should be 'key=value' format
    - https_proxy=http:127.0.0.1:8080
  replace:               // params replacer, can replace params variable before run, should be 'key=value' format, and key must start with '$'
    - $pubip=curl -s 4.ipw.cn
  log2file: true         // save program stdout to ./log/[program name].log
  enable: true           // enable autostart and timer check

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
				os.WriteFile(gocmd.JoinPathFromHere(exename+".service"), []byte(fmt.Sprintf(systemd,
					exename,
					uname,
					ugrp,
					gocmd.GetExecDir(),
					gocmd.GetExecFullpath(),
					gocmd.GetExecFullpath(),
					gocmd.GetExecFullpath(),
					psock,
				)), 0o664)
				println(fmt.Sprintf(`create systemd service file done.
1. copy "%[1]s.service" to "/etc/systemd/system/"
2. run "systemctl daemon-reload" as root
3. run "systemctl start %[1]s && systemctl enable %[1]s" as root`, exename))
				return 0
			},
		}).
		AddCommand(&gocmd.Command{
			Name:     "addpath",
			Descript: "add `pwd` to PATH env",
			RunWithExitCode: func(pi *gocmd.ProcInfo) int {
				err := gocmd.AddPathEnvFromHere()
				if err != nil {
					println(err.Error())
					return 1
				}
				println("add PATH done")
				return 0
			},
		}).
		AfterStop(func() {
			os.Remove(psock)
		}).
		BeforeStart(func() {
			time.Sleep(time.Millisecond * 500)
			if gocmd.IsExist(psock) {
				os.Remove(psock)
			}
		})
	app.ExecuteDefault("start")
	// 初始化
	if !gocmd.IsExist(cnfdir) {
		os.Mkdir(cnfdir, 0o775)
	}
	if !gocmd.IsExist(piddir) {
		os.Mkdir(piddir, 0o775)
	}
	if !gocmd.IsExist(logdir) {
		os.Mkdir(logdir, 0o775)
	}
	if gocmd.IsExist(confileOld) {
		os.Rename(confileOld, confile)
	}
	allconf = model.NewCnf(cnfdir)
	allconf.ConverFromOld()
	allconf.FromFiles()

	chrecv := make(chan *unixClient)
	// 后台处理
	if !*nokeepalive {
		go loopfunc.LoopFunc(func(params ...any) {
			td := time.Second * time.Duration(min(max(toolbox.String2Int(os.Getenv("SSDCTLD_CHECK_SECONDS"), 10), 60), 600))
			t := time.NewTicker(td)
			for {
				select {
				case <-t.C:
					// 检查所有enable==true && manualStop==false的服务状态
					allconf.ForEach(func(key string, value *model.ServiceParams) bool {
						if !value.Enable || value.ManualStop {
							return true
						}
						if _, _, ok := svrIsRunning(value); ok {
							return true
						}
						s, _ := startSvrFork(key, value)
						stdlog.Info(key + " not running, restart... " + s)
						return true
					})
				case cli := <-chrecv:
					t.Stop()
					recv(cli)
					t.Reset(td)
				}
			}
		}, "recv", nil) // stdlog.DefaultWriter())
	}
	// 开始监听
	loopfunc.LoopFunc(func(params ...any) {
		uln, err := net.ListenUnix("unix", &net.UnixAddr{Name: psock, Net: "unix"})
		if err != nil {
			stdlog.Error("listen from unix socket error: " + err.Error())
			app.Exit(1)
		}
		uln.SetUnlinkOnClose(true)
		stdlog.Info("start listen from unix socket:" + psock)
		// 监听客户端
		for {
			fd, err := uln.AcceptUnix()
			if err != nil {
				if strings.Contains(err.Error(), net.ErrClosed.Error()) {
					stdlog.Error("listener close")
					panic(fmt.Errorf("listener close"))
				}
				continue
			}
			chrecv <- &unixClient{
				conn: fd,
				buf:  make([]byte, 2048),
			}
			// recv(&unixClient{
			// 	conn: fd,
			// 	buf:  make([]byte, 2048),
			// })
		}
	}, "main proc", nil) // stdlog.DefaultWriter())
}

func recv(cli *unixClient) {
	defer func() {
		if err := recover(); err != nil {
			println("recv recover:" + err.(error).Error())
		}
		cli.conn.Close()
	}()
	for {
		cli.conn.SetReadDeadline(time.Now().Add(time.Minute))
		n, err := cli.conn.Read(cli.buf)
		if err != nil {
			// if err != io.EOF {
			println("recv error: " + err.Error())
			// }
			return
		}
		// 切割
		for _, v := range strings.Split(string(cli.buf[:n]), "\x00") {
			if len(v) == 0 {
				continue
			}
			todo := &model.ToDo{}
			err = todo.FromJSON([]byte(v))
			if err != nil {
				continue
			}
			exe, ok := allconf.GetItem(todo.Name)
			switch todo.Do {
			case model.JobClose: // 关闭
				return
			case model.JobStart: // 启动
				if !ok && todo.Name != "all" {
					cli.Send(todo.Name, "*** unknow programs: `"+todo.Name+"`")
					continue
				}
				if todo.Name == "all" {
					stdlog.Info("start all")
					allconf.ForEach(func(key string, value *model.ServiceParams) bool {
						if !value.Enable {
							return true
						}
						cli.Send(todo.Name, formatOutput(todo.Name, "STARTING...", "||> "+value.Exec+" "+strings.Join(value.Params, " "))) //"[STARTING...] "+todo.Name)
						s, _ := startSvrFork(key, value)
						cli.Send(key, s)
						return true
					})
				} else {
					cli.Send(todo.Name, formatOutput(todo.Name, "STARTING...", "||> "+exe.Exec+" "+strings.Join(exe.Params, " "))) //"[STARTING...] "+todo.Name)
					s, _ := startSvrFork(todo.Name, exe)
					cli.Send(todo.Name, s)
					stdlog.Info(s)
				}
			case model.JobStop: // 停止
				if !ok && todo.Name != "all" {
					cli.Send(todo.Name, "*** unknow programs: `"+todo.Name+"`")
					continue
				}
				if todo.Name == "all" {
					stdlog.Info("stop all")
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
					s := stopSvrFork(todo.Name, exe)
					cli.Send(todo.Name, s)
					stdlog.Warning(s)
				}
			case model.JobEnable: // 启用
				if !ok {
					cli.Send(todo.Name, "*** unknow programs: `"+todo.Name+"`")
					continue
				}
				allconf.SetEnable(todo.Name, true)
				cli.Send(todo.Name, ">>> "+todo.Name+" enabled")
				stdlog.Info("enable " + todo.Name)
			case model.JobDisable: // 停用
				if !ok {
					cli.Send(todo.Name, "*** unknow programs: `"+todo.Name+"`")
					continue
				}
				allconf.SetEnable(todo.Name, false)
				cli.Send(todo.Name, ">>> "+todo.Name+" disabled")
				stdlog.Info("disable " + todo.Name)
			case model.JobRemove: // 删除服务
				if err := allconf.DelItem(todo.Name); err != nil {
					cli.Send(todo.Name, "--- "+todo.Name+" remove failed: "+err.Error())
				} else {
					cli.Send(todo.Name, "--- "+todo.Name+" removed")
					stdlog.Info("remove " + todo.Name)
				}
			case model.JobCreate: // 新增服务
				if todo.Name == "all" ||
					todo.Name == "enable" ||
					todo.Name == "disable" ||
					todo.Name == "running" ||
					todo.Name == "stopped" {
					cli.Send("all", "can not use `"+todo.Name+"` as application's name")
					return
				}

				if err := allconf.AddItem(todo.Name, &model.ServiceParams{
					Exec:   todo.Exec,
					Params: todo.Params,
					Enable: true,
				}); err != nil {
					cli.Send(todo.Name, "+++ "+todo.Name+" add failed: "+err.Error())
				} else {
					cli.Send(todo.Name, "+++ "+todo.Name+" added")
					stdlog.Info("add " + todo.Name)
				}
			case model.JobStatus: // 状态查询
				switch todo.Name {
				case "running":
					allconf.ForEach(func(key string, value *model.ServiceParams) bool {
						_, s, ok := svrIsRunning(value)
						if ok {
							cli.Send(key, formatOutput(key, "PS", s)) //"[PS\t"+key+"]:\n"+s)
						}
						return true
					})
				case "disable":
					allconf.ForEach(func(key string, value *model.ServiceParams) bool {
						if value.Enable {
							return true
						}
						_, s, ok := svrIsRunning(value)
						if ok {
							cli.Send(key, formatOutput(key, "PS", s)) //"[PS\t"+key+"]:\n"+s)
						} else {
							cli.Send(key, formatOutput(key, "PS", "not running"))
						}
						return true
					})
				case "enable":
					allconf.ForEach(func(key string, value *model.ServiceParams) bool {
						if !value.Enable {
							return true
						}
						_, s, ok := svrIsRunning(value)
						if ok {
							cli.Send(key, formatOutput(key, "PS", s)) //"[PS\t"+key+"]:\n"+s)
						} else {
							cli.Send(key, formatOutput(key, "PS", "not running"))
						}
						return true
					})
				case "all":
					allconf.ForEach(func(key string, value *model.ServiceParams) bool {
						if value.Enable {
							cli.Send(key, statusSvr(key, value))
						}
						return true
					})
				default:
					if !ok {
						cli.Send(todo.Name, "*** unknow programs: `"+todo.Name+"`")
						continue
					}
					cli.Send(todo.Name, statusSvr(todo.Name, exe))
				}
			case model.JobList:
				switch todo.Name {
				case "enable":
					allconf.ForEach(func(key string, value *model.ServiceParams) bool {
						if value.Enable {
							cli.Send(key, listSvr(key, value))
						}
						return true
					})
				case "disable":
					allconf.ForEach(func(key string, value *model.ServiceParams) bool {
						if !value.Enable {
							cli.Send(key, listSvr(key, value))
						}
						return true
					})
				case "stopped":
					allconf.ForEach(func(key string, value *model.ServiceParams) bool {
						if value.Enable && value.ManualStop {
							cli.Send(key, listSvr(key, value))
						}
						return true
					})
				case "":
					cli.Send("", allconf.Print())
				case "all":
					allconf.ForEach(func(key string, value *model.ServiceParams) bool {
						cli.Send(key, listSvr(key, value))
						return true
					})
				default:
					if !ok {
						cli.Send(todo.Name, "*** unknow programs: `"+todo.Name+"`")
						continue
					}
					cli.Send(todo.Name, listSvr(todo.Name, exe))
				}
			case model.JobUpate: // 列出所有，刷新
				allconf.FromFiles()
				cli.Send("", allconf.Print())
			case model.JobSetLevel: // 设置优先级
				if !ok {
					cli.Send(todo.Name, "*** unknow programs: `"+todo.Name+"`")
					continue
				}
				allconf.SetLevel(todo.Name, uint32(toolbox.String2Int32(todo.Exec, 10)))
				cli.Send(todo.Name, ">>> set "+todo.Name+" start level to "+strconv.FormatUint(uint64(toolbox.String2Int32(todo.Exec, 10)), 10))
			}
		}
	}
}

func statusSvr(name string, svr *model.ServiceParams) string {
	_, ps, ok := svrIsRunning(svr)
	if !ok {
		return formatOutput(name, "PS", "not running") // "[PS\t" + name + "]:\nnot running"
	} else {
		return formatOutput(name, "PS", ps) //"[PS\t" + name + "]:\n" + ps
	}
}

func listSvr(name string, svr *model.ServiceParams) string {
	ss := strings.Builder{}
	b, err := yaml.Marshal(svr)
	if err != nil {
		return formatOutput(name, "CONFIG", "config data error, use `update` command to reload all config. "+err.Error())
	}
	ss.WriteString(formatOutput(name, "", string(b)))
	_, ps, ok := svrIsRunning(svr)
	if !ok {
		ss.WriteString(formatOutput("", "PS", "not running"))
	} else {
		ss.WriteString(formatOutput("", "PS", ps))
	}
	return ss.String()
}

func svrIsRunning(svr *model.ServiceParams) (int, string, bool) {
	if svr.Pid > 0 { // 直接查/proc信息
		b, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", svr.Pid))
		if err == nil {
			return svr.Pid, fmt.Sprintf("%d\t%s", svr.Pid, strings.ReplaceAll(string(b), "\x00", " ")), true
		}
	}
	// 遍历/proc目录寻找
	pi := model.QueryProcess(filepath.Base(svr.Exec))
	found := true
	for _, p := range pi {
		found = true
		for _, parm := range svr.Params {
			if strings.Contains(parm, "$") {
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

func startSvrFork(name string, svr *model.ServiceParams) (string, bool) {
	var spid int
	var ok bool
	var ps string
	if spid, ps, ok = svrIsRunning(svr); ok {
		if svr.Pid == 0 {
			svr.Pid = spid
		}
		return formatOutput(name, "START", "still running") + "\n" + formatOutput(name, "PS", ps), false // "[START\t" + name + "] is still running\n[PS] " + name + ":\n" + ps, false
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
	params := []string{svr.Exec} // 使用syscall时，第一个需要进程名，使用exec.cmd时不需要
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
	// 设置目录
	if svr.Dir == "" {
		svr.Dir = filepath.Dir(svr.Exec)
	}
	// 设置环境变量
	env := os.Environ()
	env = append(env, svr.Env...)
	// 开始进程
	cmd := exec.Command(svr.Exec, params[1:]...)
	cmd.Dir = svr.Dir
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		// Setsid: true,
	}
	// 设置日志输出
	var f *os.File
	if svr.Log2file {
		f, _ = os.OpenFile(filepath.Join(logdir, name+".log"), os.O_CREATE|os.O_WRONLY, 0o664)
	}
	if f != nil {
		cmd.Stdout = f
		cmd.Stderr = f
	}
	// 开始执行
	err = cmd.Start()
	if err != nil {
		return formatOutput(name, "START", "error: "+err.Error()+" '"+svr.Exec+"'"), false // "[START\t" + name + "] error: " + err.Error() + " '" + svr.Exec + "'", false
	}
	go func(pid int) {
		cmd.Wait()
		// syscall.Wait4(pid, nil, 0, nil)
	}(pid)
	time.Sleep(time.Second * time.Duration(svr.StartSec))
	pid = cmd.Process.Pid
	if !model.ProcessExist(pid) {
		spid, _, ok = svrIsRunning(svr)
		if !ok {
			return formatOutput(name, "START", "failed"), false // + "\n" + formatOutput(name, "CMD", svr.Exec+" "+strings.Join(svr.Params, " ")), false // "[START\t" + name + "] failed" + "\n[CMD\t" + name + "]:\n" + svr.Exec + " " + strings.Join(svr.Params, " "), false
		}
		pid = spid
	}
	svr.ManualStop = false
	svr.Pid = pid
	os.WriteFile(filepath.Join(piddir, name+".pid"), fmt.Appendf([]byte{}, "%d", pid), 0o664)
	return formatOutput(name, "START", "done, PID: "+strconv.Itoa(pid)), true // "[START\t" + name + "]\ndone. PID: " + fmt.Sprintf("%d", pid) + "\n[CMD] " + svr.Exec + " " + strings.Join(svr.Params, " "), true
}

func stopSvrFork(name string, svr *model.ServiceParams) string {
	pid, _, ok := svrIsRunning(svr)
	if !ok {
		return formatOutput(name, "STOP", "not running") //"[STOP\t" + name + "]:\nnot running"
	}

	err := syscall.Kill(pid, syscall.SIGINT)
	if err != nil {
		return formatOutput(name, "STOP", "error: "+err.Error()) //"[STOP\t" + name + "] error:\n" + err.Error()
	}
	// if svr.Pid == 0 {
	// 	go func(pid int) {
	// 		syscall.Wait4(pid, nil, syscall.WNOHANG, nil)
	// 	}(pid)
	// }
	for i := 0; i < 7; i++ {
		time.Sleep(time.Millisecond * 500)
		if !model.ProcessExist(pid) {
			goto GOON
		}
	}
	syscall.Kill(pid, syscall.SIGKILL)
GOON:
	svr.ManualStop = true
	svr.Pid = 0
	os.Remove(filepath.Join(piddir, name+".pid"))
	time.Sleep(time.Millisecond * 200)
	return formatOutput(name, "STOP", "done, PID: "+fmt.Sprintf("%d", pid)) // "[STOP\t" + name + "]:\ndone, PID: " + fmt.Sprintf("%d", pid)
}

func formatOutput(name, do, body string) string {
	s := ""
	if name == "" {
		if do == "" {
			return "\n" + body
		}
		s = "< " + do + " >"
	} else {
		if do == "" {
			s = "[ " + name + " ]"
		} else {
			s = "[ " + name + ":  " + do + " ]"
		}
	}
	if body == "" {
		return s
	}
	return s + "\n" + body
}
