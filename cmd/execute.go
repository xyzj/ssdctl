/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Use 'start-stop-daemon' to start a service",
	Long: `Use 'start-stop-daemon' to start a service process in the background, 
and the service startup parameters are stored in the 'luwakctl.yaml' file.
The pid file stored in /tmp.
For Example:
	luwakctl start backend sslrenew	--> Start backend and sslrenew, even if they are not enabled
	luwakctl start all	--> Start all enabled services`,
	Run: func(cmd *cobra.Command, args []string) {
		runNames(getNames(cmd, args), true, false)
	},
}
var startallCmd = &cobra.Command{
	Use:   "all",
	Short: "Start all enabled services",
	Long:  `Start all enabled services`,
	Run: func(cmd *cobra.Command, args []string) {
		runNames(confirmAll(cmd), true, false)
	},
}

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Use 'start-stop-daemon' to stop a service",
	Long: `Use 'start-stop-daemon' to stop a background service process, the pid of the service is recorded in the /tmp/[name].pid file.
For Example:
	luwakctl stop backend sslrenew	--> Stop backend and sslrenew
	luwakctl stop all	--> Stop all enabled services`,
	Run: func(cmd *cobra.Command, args []string) {
		runNames(getNames(cmd, args), false, true)
	},
}
var stopallCmd = &cobra.Command{
	Use:   "all",
	Short: "Stop all enabled services",
	Long:  `Stop all enabled services`,
	Run: func(cmd *cobra.Command, args []string) {
		runNames(confirmAll(cmd), false, true)
	},
}

// restartCmd represents the restart command
var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Use 'start-stop-daemon' to restart a service",
	Long: `Use 'start-stop-daemon' to restart a service process in the background, 
and the service startup parameters are stored in the 'luwakctl.yaml' file.
The pid file stored in /tmp.
For Example:
	luwakctl restart backend sslrenew	--> Restart backend and sslrenew
	luwakctl restart all	--> Restart all enabled services`,
	Run: func(cmd *cobra.Command, args []string) {
		runNames(getNames(cmd, args), true, true)
	},
}
var restartallCmd = &cobra.Command{
	Use:   "all",
	Short: "Restart all enabled services",
	Long:  `Restart all enabled services`,
	Run: func(cmd *cobra.Command, args []string) {
		runNames(confirmAll(cmd), true, true)
	},
}

// statusCmd 查看状态
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "View process information",
	Long:  `View process information`,
	Run: func(cmd *cobra.Command, args []string) {
		//ps h -p $(cat /tmp/backend.pid)
		var cmdd *exec.Cmd
		params := []string{"u"}
		for _, v := range getNames(cmd, args) {
			cmdd = exec.Command("cat", "/tmp/"+v+".pid")
			if b, _ := cmdd.CombinedOutput(); !bytes.HasPrefix(b, []byte("cat")) {
				params = append(params, "-p"+strings.ReplaceAll(string(b), "\n", ""))
			} else {
				println("*** service " + v + " does not exist")
			}
		}
		cmdd = exec.Command("ps", params...)
		b, err := cmdd.CombinedOutput()
		if err != nil {
			println(err.Error() + " " + string(b))
			return
		}
		println("\n" + string(b))
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(statusCmd)

	// start
	startCmd.Flags().StringSliceP("name", "n", []string{}, "Select the name of the service to start, you can set multiple parameters, such as: '-n ecms -n nboam'.")
	// startCmd.MarkFlagRequired("name")
	// start all
	startCmd.AddCommand(startallCmd)
	startallCmd.Flags().BoolP("yes", "y", false, "Don't ask if you are sure")
	// stop
	stopCmd.Flags().StringSliceP("name", "n", []string{}, "Select the name of the service to stop, you can set multiple parameters, such as: '-n ecms -n nboam'.")
	// stopCmd.MarkFlagRequired("name")
	// stop all
	stopCmd.AddCommand(stopallCmd)
	stopallCmd.Flags().BoolP("yes", "y", false, "Don't ask if you are sure")
	// restart
	restartCmd.Flags().StringSliceP("name", "n", []string{}, "Select the name of the service to restart, you can set multiple parameters, such as: '-n ecms -n nboama'.")
	// restartCmd.MarkFlagRequired("name")
	//restart
	restartCmd.AddCommand(restartallCmd)
	restartallCmd.Flags().BoolP("yes", "y", false, "Don't ask if you are sure")
}

func confirmAll(cmd *cobra.Command) []string {
	if confirm, _ := cmd.Flags().GetBool("yes"); !confirm {
		println("Please confirm if you want to do this for all services?(y/n)")
		inputreader := bufio.NewReader(os.Stdin)
		input, _ := inputreader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(input)) != "y" {
			return []string{}
		}
	}
	var names = make([]string, 0)
	for k, v := range listSvr {
		if v.Enable {
			names = append(names, k)
		}
	}
	return names
}

func getNames(cmd *cobra.Command, args []string) []string {
	names, _ := cmd.Flags().GetStringSlice("name")
	if len(names) == 0 && len(args) == 0 {
		println("Need to enter the service name")
		return []string{}
	}
	if len(names) == 0 {
		names = args
	}
	return names
}
func runNames(names []string, start, stop bool) {
	for _, name := range names {
		if svr, ok := listSvr[name]; !ok {
			println("*** service " + name + " does not exist")
			continue
		} else {
			if stop {
				stopSvr(name, svr)
				time.Sleep(time.Second)
			}
			if start {
				startSvr(name, svr)
				time.Sleep(time.Second)
			}
		}
	}
}
func startSvr(name string, svr *serviceParams) {
	dir, _ := filepath.Split(svr.Exec)
	params := []string{"--start", "--background", "-d", dir, "-m", "-p", "/tmp/" + name + ".pid", "--exec", svr.Exec, "--"}
	params = append(params, svr.Params...)
	cmd := exec.Command("start-stop-daemon", params...)
	cmd.Dir, _ = filepath.Split(svr.Exec)
	b, err := cmd.CombinedOutput()
	if err != nil {
		println("*** start " + name + " error: " + err.Error() + "\n" + string(b))
		return
	}
	// if len(b) > 0 {
	println(">>> start " + name + " done. " + string(b))
	// time.Sleep(time.Second)
	// }
}

func stopSvr(name string, svr *serviceParams) {
	params := []string{"--stop", "-p", "/tmp/" + name + ".pid"}
	cmd := exec.Command("start-stop-daemon", params...)
	cmd.Dir, _ = filepath.Split(svr.Exec)
	b, err := cmd.CombinedOutput()
	if err != nil {
		println("*** stop " + name + " error: " + err.Error() + "\n" + string(b))
		return
	}
	// if len(b) > 0 {
	println("||| stop " + name + " done. " + string(b))
	// 	time.Sleep(time.Second)
	// }
}
