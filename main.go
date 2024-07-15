package main

import (
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
	nokeepalive = flag.Bool("stopka", false, "do not check and keep programs alive")
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
	if !strings.HasSuffix(s, "\x00") {
		s += "\x00"
	}
	uc.conn.Write([]byte(s))
}

func main() {
	app = gocmd.DefaultProgram(&gocmd.Info{
		Title: "programs managerment",
		Ver:   version,
		Descript: `use "start-stop-daemon" to manager process

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
				os.WriteFile(pathtool.JoinPathFromHere(exename+".service"), []byte(fmt.Sprintf(systemd,
					exename,
					uname,
					ugrp,
					pathtool.GetExecDir(),
					pathtool.GetExecFullpath(),
					pathtool.GetExecFullpath(),
					pathtool.GetExecFullpath(),
					psock,
				)), 0o664)
				println(fmt.Sprintf(`create systemd service file done.
1. copy "%[1]s.service" to "/etc/systemd/system/"
2. run "systemctl daemon-reload" as root
3. run "systemctl start %[1]s && system enable %[1]s" as root`, exename))
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
	// 初始化
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
	allconf.ConverFromOld()
	allconf.FromFiles()

	chrecv := make(chan *unixClient)
	// 后台处理
	if !*nokeepalive {
		go loopfunc.LoopFunc(func(params ...interface{}) {
			td := time.Minute
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
						startSvrFork(key, value)
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
	loopfunc.LoopFunc(func(params ...interface{}) {
		uln, err := net.ListenUnix("unix", &net.UnixAddr{Name: psock, Net: "unix"})
		if err != nil {
			println("listen from unix socket error: " + err.Error())
			app.Exit(1)
		}
		uln.SetUnlinkOnClose(true)

		// 监听客户端
		for {
			fd, err := uln.AcceptUnix()
			if err != nil {
				if strings.Contains(err.Error(), net.ErrClosed.Error()) {
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
					cli.Send(todo.Name, "[START] unknow server name: "+todo.Name)
					continue
				}
				if todo.Name == "all" {
					allconf.ForEach(func(key string, value *model.ServiceParams) bool {
						if !value.Enable {
							return true
						}
						cli.Send(key, "[STARTING...] "+key)
						s, _ := startSvrFork(key, value)
						cli.Send(key, s)
						// if ok {
						// 	// time.Sleep(time.Second * 1)
						// 	cli.Send(key, statusSvr(key, value))
						// }
						return true
					})
				} else {
					cli.Send(todo.Name, "[STARTING...] "+todo.Name)
					s, _ := startSvrFork(todo.Name, exe)
					cli.Send(todo.Name, s)
					// if ok {
					// 	// time.Sleep(time.Second * 1)
					// 	cli.Send(todo.Name, statusSvr(todo.Name, exe))
					// }
				}
			case model.JobStop: // 停止
				if !ok && todo.Name != "all" {
					cli.Send(todo.Name, "[STOP] unknow server name: "+todo.Name)
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
				if err := allconf.DelItem(todo.Name); err != nil {
					cli.Send(todo.Name, "--- "+todo.Name+" remove failed: "+err.Error())
				} else {
					cli.Send(todo.Name, "--- "+todo.Name+" removed")
				}
			case model.JobCreate: // 新增服务
				if todo.Name == "all" {
					cli.Send("all", "can not use 'all' as application's name")
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
			return svr.Pid, fmt.Sprintf("%d\t%s", svr.Pid, strings.ReplaceAll(string(b), "\x00", " ")), true
		}
	}
	// 遍历/proc目录寻找
	pi := model.QueryProcess(filepath.Base(svr.Exec))
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
		return "[START] " + name + " error: " + err.Error() + " '" + svr.Exec + "'", false
	}
	pid = cmd.Process.Pid
	go func(pid int) {
		cmd.Wait()
		// syscall.Wait4(pid, nil, 0, nil)
	}(pid)
	time.Sleep(time.Second * time.Duration(svr.StartSec))
	if !model.ProcessExist(pid) {
		return "[START] " + name + " failed", false
	}
	svr.ManualStop = false
	svr.Pid = pid
	os.WriteFile(filepath.Join(piddir, name+".pid"), []byte(fmt.Sprintf("%d", pid)), 0o664)
	return "[START] " + name + " done, PID: " + fmt.Sprintf("%d", pid) + "\n[CMD] " + svr.Exec + " " + strings.Join(svr.Params, " "), true
}

func stopSvrFork(name string, svr *model.ServiceParams) string {
	pid, _, ok := svrIsRunning(svr)
	if !ok {
		return "[STOP] " + name + " is not running"
	}

	err := syscall.Kill(pid, syscall.SIGINT)
	if err != nil {
		return "[STOP] " + name + " error: " + err.Error()
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
	return "[STOP] " + name + " done, PID: " + fmt.Sprintf("%d", pid)
}
