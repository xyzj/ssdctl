/*
Copyright © 2021 X.Yuan

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
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/spf13/viper"
)

var (
	listSvr  = make(map[string]*serviceParams)
	yamlfile = filepath.Join(getExecDir(), "extsvr.yaml")
)

type serviceParams struct {
	Enable bool     `yaml:enable`
	Exec   string   `yaml:"exec"`
	Params []string `yaml:"params"`
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "luwakctl",
	Short: "project luwak service manager",
	Long: `project luwak micro-services manager.
You can create,delete,enable,disable,start,stop,restart all services.
This program use 'extsvr.yaml' to save the service config.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.extsvr.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	mycnf := viper.New()
	mycnf.SetConfigFile(yamlfile)
	mycnf.SetConfigType("yaml")
	// If a config file is found, read it in.
	if err := mycnf.ReadInConfig(); err == nil {
		keys := mycnf.GetStringMap("list")
		for v := range keys {
			listSvr[v] = &serviceParams{
				Enable: mycnf.GetBool("list." + v + ".enable"),
				Exec:   mycnf.GetString("list." + v + ".exec"),
				Params: mycnf.GetStringSlice("list." + v + ".params"),
			}
		}
	} else {
		// 初始化一个
		saveSvrList()
	}
}

func saveSvrList() {
	mycnf := viper.New()
	mycnf.SetConfigFile(yamlfile)
	mycnf.SetConfigType("yaml")
	mycnf.Set("list", listSvr)
	mycnf.WriteConfig()
}

func getExecDir() string {
	a, _ := os.Executable()
	execdir := filepath.Dir(a)
	if strings.Contains(execdir, "go-build") {
		execdir, _ = filepath.Abs(".")
	}
	return execdir
}
func isExist(p string) bool {
	_, err := os.Stat(p)
	return err == nil || os.IsExist(err)
}
