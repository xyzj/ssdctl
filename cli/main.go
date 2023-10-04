package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/xyzj/gopsu/gocmd"
	"github.com/xyzj/gopsu/pathtool"
)

var (
	sendfmt = "%d|%s|%s|%s|"
	psock   = pathtool.JoinPathFromHere("extsvr.sock")
	version = "0.0.0"
)

func init() {
	os.Setenv(fmt.Sprintf("%s_NOT_PARSE_FLAG", strings.ToUpper(pathtool.GetExecNameWithoutExt())), "1")
}

// var (
// 	ver  = flag.Bool("version", false, "print version number and exit")
// 	help = flag.Bool("help", false, "print help infomation")
// )

//	func printHelp() {
//		println("Usage:\n\tluwakctl start|stop|enable|disable|status|init|list|restart|create|remove|update [params...]")
//	}
func buildData(do int, name, exec, param string) []byte {
	return []byte(fmt.Sprintf(sendfmt, do, name, exec, param))
}

// 接收消息格式： fmt.Sprintf("%d|%s|%s|%s|",do,name,exec,params)
// name: 服务名称
// do: 固定1字符 0-关闭链接，1-启动，2-停止，3-启用，4-停用，5-查询状态，6-删除服务配置，7-新增服务配置，8-初始化一个文件，9-列出所有配置，10-重启指定服务，98-刷新配置，99-停止
// exec: 要执行的文件完整路径（仅新增时有效）
// params：要执行的参数，多个参数用`，`分割，（仅新增时有效）
func main() {
	gocmd.NewProgram(&gocmd.Info{
		Ver: version,
	}).AddCommand(&gocmd.Command{
		Name:     "init",
		Descript: "init a new extsvr.yaml",
		RunWithExitCode: func(pi *gocmd.ProcInfo) int {
			send2svr(os.Args[1:]...)
			return 0
		},
	}).AddCommand(&gocmd.Command{
		Name:     "shutdown",
		Descript: "shutdown the extsvr server",
		RunWithExitCode: func(pi *gocmd.ProcInfo) int {
			send2svr(os.Args[1:]...)
			return 0
		},
	}).AddCommand(&gocmd.Command{
		Name:     "start",
		Descript: "start a program",
		RunWithExitCode: func(pi *gocmd.ProcInfo) int {
			send2svr(os.Args[1:]...)
			return 0
		},
	}).AddCommand(&gocmd.Command{
		Name:     "stop",
		Descript: "stop the program",
		RunWithExitCode: func(pi *gocmd.ProcInfo) int {
			send2svr(os.Args[1:]...)
			return 0
		},
	}).AddCommand(&gocmd.Command{
		Name:     "restart",
		Descript: "restart the program",
		RunWithExitCode: func(pi *gocmd.ProcInfo) int {
			send2svr(os.Args[1:]...)
			return 0
		},
	}).AddCommand(&gocmd.Command{
		Name:     "enable",
		Descript: "set a program to autorun",
		RunWithExitCode: func(pi *gocmd.ProcInfo) int {
			send2svr(os.Args[1:]...)
			return 0
		},
	}).AddCommand(&gocmd.Command{
		Name:     "disable",
		Descript: "set a program not to autorun",
		RunWithExitCode: func(pi *gocmd.ProcInfo) int {
			send2svr(os.Args[1:]...)
			return 0
		},
	}).AddCommand(&gocmd.Command{
		Name:     "status",
		Descript: "check the status of a program",
		RunWithExitCode: func(pi *gocmd.ProcInfo) int {
			send2svr(os.Args[1:]...)
			return 0
		},
	}).AddCommand(&gocmd.Command{
		Name:     "remove",
		Descript: "remove a program config from extsvr.yaml",
		RunWithExitCode: func(pi *gocmd.ProcInfo) int {
			send2svr(os.Args[1:]...)
			return 0
		},
	}).AddCommand(&gocmd.Command{
		Name:     "create",
		Descript: "add a program config to extsvr.yaml",
		HelpMsg:  fmt.Sprintf("Usage:\n\t " + os.Args[0] + " create appname execpath param1 param2 ..."),
		RunWithExitCode: func(pi *gocmd.ProcInfo) int {
			send2svr(os.Args[1:]...)
			return 0
		},
	}).AddCommand(&gocmd.Command{
		Name:     "list",
		Descript: "print extsvr.yaml",
		RunWithExitCode: func(pi *gocmd.ProcInfo) int {
			send2svr(os.Args[1:]...)
			return 0
		},
	}).AddCommand(&gocmd.Command{
		Name:     "update",
		Descript: "update a program's config to extsvr.yaml",
		RunWithExitCode: func(pi *gocmd.ProcInfo) int {
			send2svr(os.Args[1:]...)
			return 0
		},
	}).AddCommand(&gocmd.Command{
		Name:     "shutdown",
		Descript: "shutdown the server",
		RunWithExitCode: func(pi *gocmd.ProcInfo) int {
			send2svr(os.Args[1:]...)
			return 0
		},
	}).ExecuteRun()
}

func send2svr(params ...string) {
	l := len(params)
	// 先进行一轮参数合法判断
	switch params[0] {
	case "init", "shutdown":
		print("Are you sure you want to do this?(y/n)")
		var choice string
		if _, err := fmt.Scanf("%s", &choice); err != nil {
			return
		}
		if choice != "y" {
			return
		}
	case "start", "stop", "enable", "disable", "restart":
		if l < 2 {
			println("Usage:\n\t " + os.Args[0] + " " + params[0] + " app1 app2 ...")
			return
		}
	case "status", "remove":
		if l < 2 {
			println("Usage:\n\t " + os.Args[0] + " " + params[0] + " app")
			return
		}
	case "create":
		if l < 3 {
			println("Usage:\n\t " + os.Args[0] + " create appname execpath param1 param2 ...")
			return
		}
	}
	// 连接unix socket
	conn, err := net.Dial("unix", psock)
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
			n, err := conn.Read(buf)
			if err != nil {
				if err == io.EOF || strings.Contains(err.Error(), net.ErrClosed.Error()) || strings.Contains(err.Error(), "connection reset by peer") {
					return
				}
				println(err.Error())
				return
			}
			println(string(buf[:n]))
		}
	}()
	if params[0] == "test" {
		x := buildData(10, params[1], "", "")
		x = append(x, buildData(10, params[2], "", "")...)
		x = append(x, buildData(0, "", "", "")...)
		conn.Write(x)
	}
	// 处理命令
	switch params[0] {
	case "start":
		for _, v := range params[1:] {
			conn.Write(buildData(1, v, "", ""))
			time.Sleep(time.Millisecond * 100)
		}
	case "stop":
		for _, v := range params[1:] {
			conn.Write(buildData(2, v, "", ""))
			time.Sleep(time.Millisecond * 100)
		}
	case "enable":
		for _, v := range params[1:] {
			conn.Write(buildData(3, v, "", ""))
			time.Sleep(time.Millisecond * 100)
		}
	case "disable":
		for _, v := range params[1:] {
			conn.Write(buildData(4, v, "", ""))
			time.Sleep(time.Millisecond * 100)
		}
	case "status":
		conn.Write(buildData(5, params[1], "", ""))
	case "remove":
		conn.Write(buildData(6, params[1], "", ""))
	case "create":
		conn.Write(buildData(7, params[1], params[2], strings.Join(params[3:], " ")))
	case "init":
		conn.Write(buildData(8, "", "", ""))
	case "list":
		conn.Write(buildData(9, "", "", ""))
	case "restart":
		for _, v := range params[1:] {
			conn.Write(buildData(10, v, "", ""))
			time.Sleep(time.Millisecond * 100)
		}
	case "update":
		conn.Write(buildData(98, "", "", ""))
	case "shutdown":
		conn.Write(buildData(99, "", "", ""))
	}
	time.Sleep(time.Millisecond * 100)
	conn.Write(buildData(0, "", "", ""))
	locker.Wait()
}
