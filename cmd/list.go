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
	"fmt"

	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured services",
	Long: `List all configured services, including enabled and disabled.
For Example:
	luwakctl list			--> List all
	luwakctl list enable	--> Just list enabled services
	luwakctl list disable	--> Just list disabled services`,
	Run: func(cmd *cobra.Command, args []string) {
		for k, v := range listSvr {
			println(fmt.Sprintf("Service:\t%s\n    Exec:\t%s\n    Params:\t%v\n    Enable:\t%v", k, v.Exec, v.Params, v.Enable))
			println("-------")
		}
	},
}
var listCmdEnable = &cobra.Command{
	Use:   "enable",
	Short: "List all configured services enabled",
	Long:  `List all configured services enabled`,
	Run: func(cmd *cobra.Command, args []string) {
		for k, v := range listSvr {
			if v.Enable {
				println(fmt.Sprintf("Service:\t%s\n    Exec:\t%s\n    Params:\t%v\n    Enable:\t%v", k, v.Exec, v.Params, v.Enable))
				println("-------")
			}
		}
	},
}
var listCmdDisable = &cobra.Command{
	Use:   "disable",
	Short: "List all configured services disabled",
	Long:  `List all configured services disabled.`,
	Run: func(cmd *cobra.Command, args []string) {
		for k, v := range listSvr {
			if v.Enable {
				continue
			}
			println(fmt.Sprintf("Service:\t%s\n    Exec:\t%s\n    Params:\t%v\n    Enable:\t%v", k, v.Exec, v.Params, v.Enable))
			println("-------")
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.AddCommand(listCmdEnable)
	listCmd.AddCommand(listCmdDisable)
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// listCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// listCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
