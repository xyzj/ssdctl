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
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

// nginxCmd represents the nginx command
var nginxCmd = &cobra.Command{
	Use:   "nginx",
	Short: "do some nginx operation",
	Long:  `Control the operation of nginx service`,
	Run: func(cmd *cobra.Command, args []string) {
		cmdd := exec.Command("service", "nginx", "status")
		if b, err := cmdd.CombinedOutput(); err != nil {
			println(err.Error() + " " + string(b))
		} else {
			println(string(b))
		}
	},
}
var nginxStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start nginx server",
	Long:  `Start nginx server`,
	Run: func(cmd *cobra.Command, args []string) {
		cmdd := exec.Command("service", "nginx", "start")
		if b, err := cmdd.CombinedOutput(); err != nil {
			println(err.Error() + " " + string(b))
		} else {
			println("nginx start done. " + string(b))
		}
	},
}
var nginxStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop nginx server",
	Long:  `Stop nginx server`,
	Run: func(cmd *cobra.Command, args []string) {
		cmdd := exec.Command("service", "nginx", "stop")
		if b, err := cmdd.CombinedOutput(); err != nil {
			println(err.Error() + " " + string(b))
		} else {
			println("nginx stop done. " + string(b))
		}
	},
}
var nginxRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart nginx server",
	Long:  `Restart nginx server`,
	Run: func(cmd *cobra.Command, args []string) {
		cmdd := exec.Command("service", "nginx", "restart")
		if b, err := cmdd.CombinedOutput(); err != nil {
			println(err.Error() + " " + string(b))
		} else {
			println("nginx restart done. " + string(b))
		}
	},
}
var nginxReloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload nginx server",
	Long:  `Reload nginx server`,
	Run: func(cmd *cobra.Command, args []string) {
		fsrc := filepath.Join("conf", "nginx-default")
		fdst := "/etc/nginx/sites-enabled/default"
		cmdd := exec.Command("cp", "-vf", fsrc, fdst)
		b, err := cmdd.CombinedOutput()
		if err != nil {
			println(err.Error() + string(b))
			return
		}
		print(string(b))
		cmdd = exec.Command("nginx", "-s", "reload")
		if b, err := cmdd.CombinedOutput(); err != nil {
			println(err.Error() + " " + string(b))
		} else {
			println("nginx reload done. " + string(b))
		}
	},
}

func init() {
	rootCmd.AddCommand(nginxCmd)
	nginxCmd.AddCommand(nginxStartCmd)
	nginxCmd.AddCommand(nginxStopCmd)
	nginxCmd.AddCommand(nginxRestartCmd)
	nginxCmd.AddCommand(nginxReloadCmd)
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// nginxCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// nginxCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
