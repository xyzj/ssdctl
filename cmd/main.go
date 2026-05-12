package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"extsvr/model"

	gocmd "github.com/xyzj/go-cmd"
)

var (
	version   = "0.0.0"
	clilocker = sync.WaitGroup{}
	cliConn   *net.UnixConn
)

// 接收消息格式： fmt.Sprintf("%d|%s|%s|%s|",do,name,exec,params)
// name: 服务名称
// do: 固定1字符 0-关闭链接，1-启动，2-停止，3-启用，4-停用，5-查询状态，6-删除服务配置，7-新增服务配置，9-列出所有配置，10-重启指定服务，98-刷新配置
// exec: 要执行的文件完整路径（仅新增时有效）
// params：要执行的参数，多个参数用`，`分割，（仅新增时有效）
func main() {
	gocmd.NewProgram(&gocmd.Info{
		Ver:      version,
		Title:    "start stop daemon",
		Descript: "control background process",
	}).
		AddCommand(&gocmd.Command{
			Name:     "start",
			Descript: "start a program",
			RunWithExitCode: func(pi *gocmd.ProcInfo) int {
				send2svr(os.Args[1:]...)
				return 0
			},
		}).
		AddCommand(&gocmd.Command{
			Name:     "stop",
			Descript: "stop the program",
			RunWithExitCode: func(pi *gocmd.ProcInfo) int {
				send2svr(os.Args[1:]...)
				return 0
			},
		}).
		AddCommand(&gocmd.Command{
			Name:     "restart",
			Descript: "restart the program",
			RunWithExitCode: func(pi *gocmd.ProcInfo) int {
				send2svr(os.Args[1:]...)
				return 0
			},
		}).
		AddCommand(&gocmd.Command{
			Name:     "enable",
			Descript: "set a program to autorun",
			RunWithExitCode: func(pi *gocmd.ProcInfo) int {
				send2svr(os.Args[1:]...)
				return 0
			},
		}).
		AddCommand(&gocmd.Command{
			Name:     "disable",
			Descript: "set a program not to autorun",
			RunWithExitCode: func(pi *gocmd.ProcInfo) int {
				send2svr(os.Args[1:]...)
				return 0
			},
		}).
		AddCommand(&gocmd.Command{
			Name:     "remove",
			Descript: "remove a program config from cnf.d",
			RunWithExitCode: func(pi *gocmd.ProcInfo) int {
				send2svr(os.Args[1:]...)
				return 0
			},
		}).
		AddCommand(&gocmd.Command{
			Name:     "create",
			Descript: "add a program config to cnf.d",
			HelpMsg:  "Usage:\n\t " + os.Args[0] + " create appname execpath param1 param2 ...",
			RunWithExitCode: func(pi *gocmd.ProcInfo) int {
				send2svr(os.Args[1:]...)
				return 0
			},
		}).
		AddCommand(&gocmd.Command{
			Name:     "status",
			Descript: "check the status of a program",
			HelpMsg: `Usage:
  status [params...]

Available commands:
  running	show all running programs status
  enable	show all enabled programs status
  disable	show all disabled programs status
  all		show all enabled programs status
  [name]	show [name] status`,
			RunWithExitCode: func(pi *gocmd.ProcInfo) int {
				send2svr(os.Args[1:]...)
				return 0
			},
		}).
		AddCommand(&gocmd.Command{
			Name:     "list",
			Descript: "list program config and status",
			HelpMsg: `Usage:
  list [params...]

Available commands:
  enable	list all enabled programs
  disable	list all disabled programs
  stopped	list all enabled but manual stopped programs
  all		show all programs
  [name]	list [name] process config and status
  [nothing]	list all programs configured`,
			RunWithExitCode: func(pi *gocmd.ProcInfo) int {
				send2svr(os.Args[1:]...)
				return 0
			},
		}).
		AddCommand(&gocmd.Command{
			Name:     "update",
			Descript: "reload all program config in cnf.d",
			RunWithExitCode: func(pi *gocmd.ProcInfo) int {
				send2svr(os.Args[1:]...)
				return 0
			},
		}).
		AddCommand(&gocmd.Command{
			Name:     "setlevel",
			Descript: "set a program's start level, 1-255",
			RunWithExitCode: func(pi *gocmd.ProcInfo) int {
				send2svr(os.Args[1:]...)
				return 0
			},
		}).
		AddCommand(&gocmd.Command{
			Name:     "shell",
			Descript: "launch an interactive shell environment.",
			RunWithExitCode: func(pi *gocmd.ProcInfo) int {
				shell2svr()
				return 0
			},
		}).
		ExecuteNotParseFlag("shell")
}
func conn2svr() error {
	var err error
	pid := os.Getpid()
	// 连接unix socket
	cliConn, err = net.ListenUnixgram("unixgram", model.CliAddr(pid))
	if err != nil {
		return err
	}
	clilocker.Go(func() {
		buf := make([]byte, 4096)
		for {
			// conn.SetReadDeadline(time.Now().Add(time.Second * 2))
			n, _, err := cliConn.ReadFromUnix(buf)
			if err != nil {
				// if strings.Contains(err.Error(), net.ErrClosed.Error()) ||
				// 	strings.Contains(err.Error(), "connection reset by peer") ||
				// 	strings.Contains(err.Error(), "EOF") { // err == io.EOF ||
				// 	return
				// }
				println(err.Error())
				// os.Exit(1)
				return
			}
			if s := string(buf[:n]); s == "END" {
				// os.Exit(0)
				return
			} else {
				println(s)
			}
		}
	})
	return nil
}

func send2svr(params ...string) {
	if !checkParams(params) {
		return
	}
	err := conn2svr()
	if err != nil {
		println(err.Error())
		return
	}
	defer cliConn.Close()
	doJob(params)
	time.Sleep(time.Millisecond * 200)
	clo := &model.ToDo{
		Do: model.JobEnd,
	}
	cliConn.WriteToUnix(clo.ToJSON(), model.SvrAddr)
	clilocker.Wait()
}

func shell2svr() {
	printHelp := func() {
		fmt.Println(`Usage:
  [command] [args...]

Commands:
  help                                 show this help
  exit | quit                          exit interactive shell
  start app1 app2 ...                  start one or more programs
  stop app1 app2 ...                   stop one or more programs
  restart app1 app2 ...                restart one or more programs
  enable app1 app2 ...                 enable autorun for programs
  disable app1 app2 ...                disable autorun for programs
  status app|running|enable|disable|all
                                       query status
  list [name|enable|disable|stopped|all]
                                       list program config/status
  remove app                           remove one program config
  create app execpath [param1 ...]     add one program config
  update                               reload/update config in daemon
  setlevel app level(1-255)            set start level for one app`)
	}
	err := conn2svr()
	if err != nil {
		println(err.Error())
		return
	}
	defer cliConn.Close()
	fmt.Println("ssdctl interactive shell, input 'help' to show commands, 'exit' or 'quit' to quit")
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("ssdctl> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		params := strings.Fields(line)
		if len(params) == 0 {
			continue
		}

		cmd := strings.ToLower(params[0])
		switch cmd {
		case "help":
			printHelp()
			continue
		case "exit", "quit":
			clo := &model.ToDo{Do: model.JobEnd}
			cliConn.WriteToUnix(clo.ToJSON(), model.SvrAddr)
			clilocker.Wait()
			return
		}

		if !checkParams(params) {
			continue
		}
		doJob(params)
	}

	if err := scanner.Err(); err != nil {
		println(err.Error())
	}
	clo := &model.ToDo{Do: model.JobEnd}
	cliConn.WriteToUnix(clo.ToJSON(), model.SvrAddr)
	clilocker.Wait()
}

func checkParams(params []string) bool {
	if len(params) == 0 {
		return false
	}
	cmd := params[0]
	// 先进行一轮参数合法判断
	switch cmd {
	case model.NameStart, model.NameStop, model.NameEnable, model.NameDisable, model.NameRestart:
		if len(params) < 2 {
			println("Usage:\n\t " + os.Args[0] + " " + cmd + " app1 app2 ...")
			return false
		}
	case model.NameStatus, model.NameRemove:
		if len(params) < 2 {
			println("Usage:\n\t " + os.Args[0] + " " + cmd + " app")
			return false
		}
	case model.NameCreate, model.NameStartLevel:
		if len(params) < 3 {
			println("Usage:\n\t " + os.Args[0] + " create appname execpath param1 param2 ...")
			return false
		}
	}
	return true
}
func doJob(params []string) {
	// 处理命令
	switch cmd := params[0]; cmd {
	case model.NameStart:
		for _, v := range params[1:] {
			todo := &model.ToDo{
				Name: v,
				Do:   model.JobStart,
			}
			cliConn.WriteToUnix(todo.ToJSON(), model.SvrAddr)
			time.Sleep(time.Millisecond * 200)
		}
	case model.NameStop:
		for _, v := range params[1:] {
			todo := &model.ToDo{
				Name: v,
				Do:   model.JobStop,
			}
			cliConn.WriteToUnix(todo.ToJSON(), model.SvrAddr)
			time.Sleep(time.Millisecond * 200)
		}
	case model.NameRestart:
		for _, v := range params[1:] {
			todo := &model.ToDo{
				Name: v,
				Do:   model.JobStop,
			}
			cliConn.WriteToUnix(todo.ToJSON(), model.SvrAddr)
			time.Sleep(time.Millisecond * 200)
			todo = &model.ToDo{
				Name: v,
				Do:   model.JobStart,
			}
			cliConn.WriteToUnix(todo.ToJSON(), model.SvrAddr)
			time.Sleep(time.Millisecond * 200)
		}
	case model.NameEnable:
		for _, v := range params[1:] {
			todo := &model.ToDo{
				Name: v,
				Do:   model.JobEnable,
			}
			cliConn.WriteToUnix(todo.ToJSON(), model.SvrAddr)
			time.Sleep(time.Millisecond * 200)
		}
	case model.NameDisable:
		for _, v := range params[1:] {
			todo := &model.ToDo{
				Name: v,
				Do:   model.JobDisable,
			}
			cliConn.WriteToUnix(todo.ToJSON(), model.SvrAddr)
			time.Sleep(time.Millisecond * 200)
		}
	case model.NameStatus:
		todo := &model.ToDo{
			Name: params[1],
			Do:   model.JobStatus,
		}
		cliConn.WriteToUnix(todo.ToJSON(), model.SvrAddr)
		time.Sleep(time.Millisecond * 200)
	case model.NameRemove:
		todo := &model.ToDo{
			Name: params[1],
			Do:   model.JobRemove,
		}
		cliConn.WriteToUnix(todo.ToJSON(), model.SvrAddr)
		time.Sleep(time.Millisecond * 200)
	case model.NameCreate:
		todo := &model.ToDo{
			Name:   params[1],
			Do:     model.JobCreate,
			Exec:   params[2],
			Params: params[3:],
		}
		cliConn.WriteToUnix(todo.ToJSON(), model.SvrAddr)
		time.Sleep(time.Millisecond * 200)
	case model.NameList:
		var todo *model.ToDo
		if len(params) > 1 {
			todo = &model.ToDo{
				Do:   model.JobList,
				Name: params[1],
			}
		} else {
			todo = &model.ToDo{
				Do: model.JobList,
			}
		}
		cliConn.WriteToUnix(todo.ToJSON(), model.SvrAddr)
		time.Sleep(time.Millisecond * 200)
	case model.NameUpdate:
		todo := &model.ToDo{
			Do: model.JobUpate,
		}
		cliConn.WriteToUnix(todo.ToJSON(), model.SvrAddr)
		time.Sleep(time.Millisecond * 200)
	case model.NameShutdown:
		todo := &model.ToDo{
			Do: model.JobShutdown,
		}
		cliConn.WriteToUnix(todo.ToJSON(), model.SvrAddr)
		time.Sleep(time.Millisecond * 200)
	case model.NameStartLevel:
		todo := &model.ToDo{
			Name: params[1],
			Do:   model.JobSetLevel,
			Exec: params[2],
		}
		cliConn.WriteToUnix(todo.ToJSON(), model.SvrAddr)
		time.Sleep(time.Millisecond * 200)
	default:
		fmt.Printf("unknown command: %s, input 'help' to show commands\n", cmd)
	}
}
