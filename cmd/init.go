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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
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
				listSvr["bsjk"] = &serviceParams{
					Enable: true,
					Exec:   "/opt/bin/businessjk",
					Params: []string{"-portable", "-conf=bsjk.conf", "-http=6827", "-forcehttp"},
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
				listSvr["sslrenew"] = &serviceParams{
					Enable: false,
					Exec:   "/opt/bin/sslrenew",
				}
			case "hcloud":
				listSvr["sslrenew"] = &serviceParams{
					Enable: false,
					Exec:   "/home/xy/bin/sslrenew",
					Params: []string{"-debug"},
				}
				listSvr["frpcall"] = &serviceParams{
					Enable: true,
					Exec:   "/home/xy/bin/frp/frpc",
					Params: []string{"-c=/home/xy/bin/frp/frpc-all.ini"},
				}
				listSvr["frpcssh"] = &serviceParams{
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
var sqlCmd = &cobra.Command{
	Use:   "sql",
	Short: "Initialize sql database",
	Long:  `Use the script in the sql directory to initialize the database`,
	Run: func(cmd *cobra.Command, args []string) {
		os.Remove("/tmp/initdb.sh")
		os.Remove("/tmp/newdb.sql")
		host, _ := cmd.Flags().GetString("host")
		port, _ := cmd.Flags().GetString("port")
		username, _ := cmd.Flags().GetString("username")
		passwd, _ := cmd.Flags().GetString("passwd")
		dbmark, _ := cmd.Flags().GetString("dbmark")
		dir, err := ioutil.ReadDir("sql")
		if err != nil {
			println("sql files not found:" + err.Error())
		} else {
			var bb bytes.Buffer
			for _, fi := range dir {
				if !fi.IsDir() {
					b, err := ioutil.ReadFile(filepath.Join(".", "sql", fi.Name()))
					if err == nil {
						bb.Write(b)
						bb.WriteString("\n")
					} else {
						println("read " + fi.Name() + " err: " + err.Error())
					}
				}
			}
			// 替换前缀
			snew := strings.ReplaceAll(bb.String(), "v5db_", dbmark+"_")
			ioutil.WriteFile("/tmp/newdb.sql", []byte(snew), 0666)
			bb.Reset()
			bb.WriteString("#!/bin/bash\n")
			bb.WriteString(filepath.Join(getExecDir(), "mysql") + " -h" + host + " -P" + port + " -u" + username + " -p" + passwd + " < /tmp/newsql.sql\n")
			ioutil.WriteFile("/tmp/initdb.sh", bb.Bytes(), 0755)
		}
		cmdd := exec.Command("/tmp/initdb.sh")
		b, err := cmdd.CombinedOutput()
		if err != nil {
			println("init database error " + err.Error() + string(b))
			return
		}
		println("init database done. " + string(b))
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.AddCommand(sqlCmd)

	sqlCmd.Flags().StringP("host", "H", "127.0.0.1", "mariadb host")
	sqlCmd.Flags().StringP("port", "P", "3306", "mariadb port")
	sqlCmd.Flags().StringP("passwd", "p", "lp1234xy", "mariadb password")
	sqlCmd.Flags().StringP("username", "u", "root", "mariadb username")
	sqlCmd.Flags().StringP("dbmark", "m", "v5db", "mariadb database name mark, like 'v5db'")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// initCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
