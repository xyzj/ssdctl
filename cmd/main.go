package main

import (
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
	version = "0.0.0"
)

func init() {
	os.Setenv(strings.ToUpper(gocmd.GetExecNameWithoutExt())+"_NOT_PARSE_FLAG", "1")
}

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
			Descript: "remove a program config from extsvr.yaml",
			RunWithExitCode: func(pi *gocmd.ProcInfo) int {
				send2svr(os.Args[1:]...)
				return 0
			},
		}).
		AddCommand(&gocmd.Command{
			Name:     "create",
			Descript: "add a program config to extsvr.yaml",
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
			Descript: "update a program's config to extsvr.yaml",
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
		ExecuteRun()
}

func send2svr(params ...string) {
	l := len(params)
	// 先进行一轮参数合法判断
	switch params[0] {
	case model.NameStart, model.NameStop, model.NameEnable, model.NameDisable, model.NameRestart:
		if l < 2 {
			println("Usage:\n\t " + os.Args[0] + " " + params[0] + " app1 app2 ...")
			return
		}
	case model.NameStatus, model.NameRemove:
		if l < 2 {
			println("Usage:\n\t " + os.Args[0] + " " + params[0] + " app")
			return
		}
	case model.NameCreate, model.NameStartLevel:
		if l < 3 {
			println("Usage:\n\t " + os.Args[0] + " create appname execpath param1 param2 ...")
			return
		}
	}
	pid := os.Getpid()
	laddr, _ := net.ResolveUnixAddr("unixgram", fmt.Sprintf(model.CliSock, pid))
	raddr, _ := net.ResolveUnixAddr("unixgram", model.SvrSock)
	// 连接unix socket
	conn, err := net.ListenUnixgram("unixgram", laddr)
	if err != nil {
		println(err.Error())
		return
	}
	defer func() {
		conn.Close()
	}()
	locker := sync.WaitGroup{}
	locker.Add(1)
	go func() {
		defer locker.Done()
		buf := make([]byte, 4096)
		for {
			// conn.SetReadDeadline(time.Now().Add(time.Second * 2))
			n, _, err := conn.ReadFromUnix(buf)
			if err != nil {
				// if strings.Contains(err.Error(), net.ErrClosed.Error()) ||
				// 	strings.Contains(err.Error(), "connection reset by peer") ||
				// 	strings.Contains(err.Error(), "EOF") { // err == io.EOF ||
				// 	return
				// }
				println(err.Error())
				os.Exit(1)
				return
			}
			if s := string(buf[:n]); s == "END" {
				os.Exit(0)
				return
			} else {
				println(s + "\n")
			}
		}
	}()
	// 处理命令
	switch params[0] {
	case model.NameStart:
		for _, v := range params[1:] {
			todo := &model.ToDo{
				Name: v,
				Do:   model.JobStart,
			}
			conn.WriteToUnix(todo.ToJSON(), raddr)
			time.Sleep(time.Millisecond * 200)
		}
	case model.NameStop:
		for _, v := range params[1:] {
			todo := &model.ToDo{
				Name: v,
				Do:   model.JobStop,
			}
			conn.WriteToUnix(todo.ToJSON(), raddr)
			time.Sleep(time.Millisecond * 200)
		}
	case model.NameRestart:
		for _, v := range params[1:] {
			todo := &model.ToDo{
				Name: v,
				Do:   model.JobStop,
			}
			conn.WriteToUnix(todo.ToJSON(), raddr)
			time.Sleep(time.Millisecond * 200)
			todo = &model.ToDo{
				Name: v,
				Do:   model.JobStart,
			}
			conn.WriteToUnix(todo.ToJSON(), raddr)
			// todo := &model.ToDo{
			// 	Name: v,
			// 	Do:   model.JobRestart,
			// }
			// conn.Write(todo.ToJSON())
			time.Sleep(time.Millisecond * 200)
		}
	case model.NameEnable:
		for _, v := range params[1:] {
			todo := &model.ToDo{
				Name: v,
				Do:   model.JobEnable,
			}
			conn.WriteToUnix(todo.ToJSON(), raddr)
			time.Sleep(time.Millisecond * 200)
		}
	case model.NameDisable:
		for _, v := range params[1:] {
			todo := &model.ToDo{
				Name: v,
				Do:   model.JobDisable,
			}
			conn.WriteToUnix(todo.ToJSON(), raddr)
			time.Sleep(time.Millisecond * 200)
		}
	case model.NameStatus:
		todo := &model.ToDo{
			Name: params[1],
			Do:   model.JobStatus,
		}
		conn.WriteToUnix(todo.ToJSON(), raddr)
		time.Sleep(time.Millisecond * 200)
	case model.NameRemove:
		todo := &model.ToDo{
			Name: params[1],
			Do:   model.JobRemove,
		}
		conn.WriteToUnix(todo.ToJSON(), raddr)
		time.Sleep(time.Millisecond * 200)
	case model.NameCreate:
		todo := &model.ToDo{
			Name:   params[1],
			Do:     model.JobCreate,
			Exec:   params[2],
			Params: params[3:],
		}
		conn.WriteToUnix(todo.ToJSON(), raddr)
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
		conn.WriteToUnix(todo.ToJSON(), raddr)
		time.Sleep(time.Millisecond * 200)
	case model.NameUpdate:
		todo := &model.ToDo{
			Do: model.JobUpate,
		}
		conn.WriteToUnix(todo.ToJSON(), raddr)
		time.Sleep(time.Millisecond * 200)
	case model.NameShutdown:
		todo := &model.ToDo{
			Do: model.JobShutdown,
		}
		conn.WriteToUnix(todo.ToJSON(), raddr)
		time.Sleep(time.Millisecond * 200)
	case model.NameStartLevel:
		todo := &model.ToDo{
			Name: params[1],
			Do:   model.JobSetLevel,
			Exec: params[2],
		}
		conn.WriteToUnix(todo.ToJSON(), raddr)
		time.Sleep(time.Millisecond * 200)
	}
	time.Sleep(time.Millisecond * 200)
	clo := &model.ToDo{
		Do: model.JobEnd,
	}
	conn.WriteToUnix(clo.ToJSON(), raddr)
	locker.Wait()
}
