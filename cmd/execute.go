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
The pid file stored in /run.`,
	Run: func(cmd *cobra.Command, args []string) {
		names, _ := cmd.Flags().GetStringSlice("name")
		runNames(names, true, false)
	},
}
var startallCmd = &cobra.Command{
	Use:   "all",
	Short: "Start all enabled services",
	Long:  `Start all enabled services`,
	Run: func(cmd *cobra.Command, args []string) {
		var names = make([]string, 0)
		println("Please confirm whether you need to start all services?(y/n)")
		inputreader := bufio.NewReader(os.Stdin)
		input, _ := inputreader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(input)) != "y" {
			return
		}
		for k, v := range listSvr {
			if v.Enable {
				names = append(names, k)
			}
		}
		runNames(names, true, false)
	},
}

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Use 'start-stop-daemon' to stop a service",
	Long:  `Use 'start-stop-daemon' to stop a background service process, the pid of the service is recorded in the /run/[name].pid file.`,
	Run: func(cmd *cobra.Command, args []string) {
		names, _ := cmd.Flags().GetStringSlice("name")
		runNames(names, false, true)
	},
}
var stopallCmd = &cobra.Command{
	Use:   "all",
	Short: "Stop all enabled services",
	Long:  `Stop all enabled services`,
	Run: func(cmd *cobra.Command, args []string) {
		var names = make([]string, 0)
		println("Please confirm whether you need to stop all services?(y/n)")
		inputreader := bufio.NewReader(os.Stdin)
		input, _ := inputreader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(input)) != "y" {
			return
		}
		for k, v := range listSvr {
			if v.Enable {
				names = append(names, k)
			}
		}
		runNames(names, false, true)
	},
}

// restartCmd represents the restart command
var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Use 'start-stop-daemon' to restart a service",
	Long: `Use 'start-stop-daemon' to restart a service process in the background, 
and the service startup parameters are stored in the 'luwakctl.yaml' file.
The pid file stored in /run.`,
	Run: func(cmd *cobra.Command, args []string) {
		names, _ := cmd.Flags().GetStringSlice("name")
		runNames(names, true, true)
	},
}
var restartallCmd = &cobra.Command{
	Use:   "all",
	Short: "Restart all enabled services",
	Long:  `Restart all enabled services`,
	Run: func(cmd *cobra.Command, args []string) {
		var names = make([]string, 0)
		println("Please confirm whether you need to restart all services?(y/n)")
		inputreader := bufio.NewReader(os.Stdin)
		input, _ := inputreader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(input)) != "y" {
			return
		}
		for k, v := range listSvr {
			if v.Enable {
				names = append(names, k)
			}
		}
		runNames(names, true, true)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(restartCmd)

	// start
	startCmd.Flags().StringSliceP("name", "n", []string{}, "Select the name of the service to start, you can set multiple parameters, such as: '-n ecms -n nboam'.")
	startCmd.MarkFlagRequired("name")
	// start all
	startCmd.AddCommand(startallCmd)
	startallCmd.Flags().BoolP("yes", "y", false, "Don't ask if you are sure")
	// stop
	stopCmd.Flags().StringSliceP("name", "n", []string{}, "Select the name of the service to stop, you can set multiple parameters, such as: '-n ecms -n nboam'.")
	stopCmd.MarkFlagRequired("name")
	// stop all
	stopCmd.AddCommand(stopallCmd)
	stopallCmd.Flags().BoolP("yes", "y", false, "Don't ask if you are sure")
	// restart
	restartCmd.Flags().StringSliceP("name", "n", []string{}, "Select the name of the service to restart, you can set multiple parameters, such as: '-n ecms -n nboama'.")
	restartCmd.MarkFlagRequired("name")
	//restart
	restartCmd.AddCommand(restartallCmd)
	restartallCmd.Flags().BoolP("yes", "y", false, "Don't ask if you are sure")
}

func runNames(names []string, start, stop bool) {
	for _, name := range names {
		if svr, ok := listSvr[name]; !ok {
			println("*** service " + name + " does not exist")
			continue
		} else {
			if stop {
				stopSvr(name, svr)
			}
			if start {
				startSvr(name, svr)
			}
		}
	}
}
func startSvr(name string, svr *serviceParams) {
	dir, _ := filepath.Split(svr.Exec)
	params := []string{"--start", "--background", "-d", dir, "-m", "-p", "/run/" + name + ".pid", "--exec", svr.Exec, "--"}
	params = append(params, svr.Params...)
	cmd := exec.Command("start-stop-daemon", params...)
	cmd.Dir, _ = filepath.Split(svr.Exec)
	b, err := cmd.CombinedOutput()
	if err != nil {
		println("==> start " + name + " error: " + err.Error())
		return
	}
	if len(b) > 0 {
		println("==> start " + name + " : " + string(b))
		time.Sleep(time.Second)
	}
}

func stopSvr(name string, svr *serviceParams) {
	params := []string{"--stop", "-p", "/run/" + name + ".pid"}
	cmd := exec.Command("start-stop-daemon", params...)
	cmd.Dir, _ = filepath.Split(svr.Exec)
	b, err := cmd.CombinedOutput()
	if err != nil {
		println("--> stop " + name + " error: " + err.Error())
		return
	}
	if len(b) > 0 {
		println("--> stop " + name + " : " + string(b))
		time.Sleep(time.Second)
	}
}
