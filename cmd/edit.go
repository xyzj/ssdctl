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
	"github.com/spf13/cobra"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a configured service",
	Long: `Delete a configured service, the deleted service cannot be restored, but it can be recreated
For Example:
	luwakctl delete backend sslrenew`,
	Run: edit,
}

// disableCmd represents the disable command
var disableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable a configured service",
	Long: `Disable a configured service, The disabled service will NOT be executed when 'start|stop|restart all'
	For Example:
		luwakctl disable all	--> Disable all configured services
		luwakctl disable backend sslrenew	--> Just disable backend and sslrenew`,
	Run: edit,
}

// enableCmd represents the enable command
var enableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable a configured service",
	Long: `Enable a configured service, The enabled service will be executed when 'start|stop|restart all'
	For Example:
		luwakctl enable all	--> Enable all configured services
		luwakctl enable backend sslrenew	--> Just enable backend and sslrenew`,
	Run: edit,
}

func edit(cmd *cobra.Command, args []string) {
	names, _ := cmd.Flags().GetStringSlice("name")
	if len(names) == 0 && len(args) == 0 {
		println("Need to enter the service name")
		return
	}
	if len(names) == 0 {
		names = args
	}
	for _, name := range names {
		if _, ok := listSvr[name]; !ok {
			println("service " + name + " does not exist")
			continue
		}
		switch cmd.Name() {
		case "delete":
			delete(listSvr, name)
			println("service " + name + " has been deleted")
		case "disable":
			listSvr[name].Enable = false
			saveSvrList()
			println("service " + name + " disabled")
		case "enable":
			listSvr[name].Enable = true
			println("service " + name + " enabled")
		}
	}
	saveSvrList()
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(disableCmd)
	rootCmd.AddCommand(enableCmd)

	deleteCmd.Flags().StringSliceP("name", "n", []string{}, "Select the name of the service to be deleted")
	disableCmd.Flags().StringSliceP("name", "n", []string{}, "Select the name of the service to be disabled")
	enableCmd.Flags().StringSliceP("name", "n", []string{}, "Select the name of the service to be enabled")
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// deleteCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// deleteCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
