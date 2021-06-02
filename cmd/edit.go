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
	"strings"

	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize an empty configuration file, or clear the current configuration file",
	Long: `Initialize an empty configuration file, or clear the current configuration file
For Example:
	luwakctl init 		--> This will create a empty configuration file
	luwakctl init luwak --> This will create a configuration file with project luwak's micro-services`,
	Run: func(cmd *cobra.Command, args []string) {
		if isExist(yamlfile) {
			print("Are you sure you want to create a new profile? The current configuration file will be overwritten and cannot be restored(y/n)")
			inputreader := bufio.NewReader(os.Stdin)
			input, _ := inputreader.ReadString('\n')
			if strings.ToLower(strings.TrimSpace(input)) != "y" {
				return
			}
		}
		listSvr = make(map[string]*serviceParams)
		if len(args) > 0 {
			switch args[0] {
			case "luwak":
				// 默认启动的
				listSvr["sslrenew"] = &serviceParams{
					Enable: true,
					Exec:   "/opt/bin/sslrenew",
				}
				listSvr["backend"] = &serviceParams{
					Enable: true,
					Exec:   "/opt/bin/backend",
					Params: []string{"-portable", "-conf=backend.conf", "-http=6819", "-forcehttp"},
				}
				listSvr["uas"] = &serviceParams{
					Enable: true,
					Exec:   "/opt/bin/uas",
					Params: []string{"-portable", "-conf=uas.conf", "-http=6820", "-forcehttp"},
				}
				listSvr["ecms-mod"] = &serviceParams{
					Enable: true,
					Exec:   "/opt/bin/ecms-mod",
					Params: []string{"-portable", "-conf=ecms.conf", "-http=6821", "-tcp=6828", "-tcpmodule=wlst", "-forcehttp"},
				}
				listSvr["task"] = &serviceParams{
					Enable: true,
					Exec:   "/opt/bin/task",
					Params: []string{"-portable", "-conf=task.conf", "-http=6822", "-forcehttp"},
				}
				listSvr["logger"] = &serviceParams{
					Enable: true,
					Exec:   "/opt/bin/logger",
					Params: []string{"-portable", "-conf=logger.conf", "-http=6823", "-forcehttp"},
				}
				listSvr["msgpush"] = &serviceParams{
					Enable: true,
					Exec:   "/opt/bin/msgpush",
					Params: []string{"-portable", "-conf=msgpush.conf", "-http=6824", "-forcehttp"},
				}
				listSvr["asset"] = &serviceParams{
					Enable: true,
					Exec:   "/opt/bin/assetmanager",
					Params: []string{"-portable", "-conf=asset.conf", "-http=6825", "-forcehttp"},
				}
				listSvr["gis"] = &serviceParams{
					Enable: true,
					Exec:   "/opt/bin/gismanager",
					Params: []string{"-portable", "-conf=gis.conf", "-http=6826", "-forcehttp"},
				}
				listSvr["uiact"] = &serviceParams{
					Enable: true,
					Exec:   "/opt/bin/netcore/userinteraction",
					Params: []string{"--log=/opt/bin/log/userinteraction", "--conf=/opt/bin/conf/userinteraction"},
				}
				listSvr["dpwlst"] = &serviceParams{
					Enable: true,
					Exec:   "/opt/bin/netcore/dataparser-wlst",
					Params: []string{"--log=/opt/bin/log/dataparser-wlst", "--conf=/opt/bin/conf/dataparser-wlst"},
				}
				listSvr["dm"] = &serviceParams{
					Enable: true,
					Exec:   "/opt/bin/netcore/datamaintenance",
					Params: []string{"--log=/opt/bin/log/datamaintenance", "--conf=/opt/bin/conf/datamaintenance"},
				}
				// 默认不起动的
				listSvr["nboam"] = &serviceParams{
					Enable: false,
					Exec:   "/opt/bin/nboam",
					Params: []string{"-portable", "-conf=nboam.conf", "-http=6835", "-forcehttp"},
				}
				listSvr["ftpupg"] = &serviceParams{
					Enable: false,
					Exec:   "/opt/bin/ftpupgrade",
					Params: []string{"-portable", "-conf=ftp.conf", "-http=6829", "-ftp=6830", "-forcehttp"},
				}
				listSvr["dpnb"] = &serviceParams{
					Enable: false,
					Exec:   "/opt/bin/netcore/dataparser-nbiot",
					Params: []string{"--log=/opt/bin/log/dataparser-nbiot", "--conf=/opt/bin/conf/dataparser-nbiot"},
				}
			case "hcloud":
				listSvr["sslrenew"] = &serviceParams{
					Enable: true,
					Exec:   "/home/xy/bin/sslrenew",
				}
				listSvr["frpcall"] = &serviceParams{
					Enable: true,
					Exec:   "/home/xy/bin/frp/frpc",
					Params: []string{"-c=/home/xy/bin/frp/frpc-all.ini"},
				}
				listSvr["frpc"] = &serviceParams{
					Enable: true,
					Exec:   "/home/xy/bin/frp/frpc",
					Params: []string{"-c=/home/xy/bin/frp/frpc.ini"},
				}
				listSvr["watchme"] = &serviceParams{
					Enable: true,
					Exec:   "/home/xy/bin/frp/watchme",
				}
				listSvr["hcloud"] = &serviceParams{
					Enable: true,
					Exec:   "/home/xy/bin/hcloud",
					Params: []string{"-conf=/home/xy/bin/hcloud.conf", "-https=2087", "-http=6820"},
				}
			}
		}
		saveSvrList()
		println("Configuration file initialization completed")
	},
}

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
		case "init":
			listSvr = make(map[string]*serviceParams)
			saveSvrList()
		}
	}
	saveSvrList()
}

func init() {
	rootCmd.AddCommand(initCmd)
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
