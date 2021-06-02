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
	"strings"

	"github.com/spf13/cobra"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new service",
	Long: `Create a new service, if the service name already exists, it cannot be created
For Example:
	luwakctl create -n newsvr -e /opt/bin/newsvr -p "-conf=./newsvr.conf"`,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		if strings.TrimSpace(name) == "all" {
			println("you can not use this name")
			return
		}
		// 检查重名
		if _, ok := listSvr[name]; ok {
			println("service " + name + " already exists")
			return
		}
		// 添加服务
		listSvr[name] = &serviceParams{
			Enable: true,
			Exec: func() string {
				exec, _ := cmd.Flags().GetString("exec")
				return exec
			}(),
			Params: func() []string {
				if plst, err := cmd.Flags().GetStringSlice("param"); err != nil {
					return []string{}
				} else {
					return plst
				}
			}(),
		}
		saveSvrList()
		println("service " + name + " create success")
	},
}

func init() {
	rootCmd.AddCommand(createCmd)

	// 启动参数
	createCmd.Flags().StringP("name", "n", "", "Set the name of the service to be created")
	createCmd.MarkFlagRequired("name")
	createCmd.Flags().StringP("exec", "e", "", "Set service execution path")
	createCmd.MarkFlagRequired("exec")
	createCmd.Flags().StringSliceP("param", "p", []string{}, `Set the startup configuration parameters of the service, you can set multiple parameters, such as:'--param "a=1" --param "b=2"'`)
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// createCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// createCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
