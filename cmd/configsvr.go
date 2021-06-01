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
	"bytes"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// configsvrCmd represents the configsvr command
var configsvrCmd = &cobra.Command{
	Use:   "configsvr",
	Short: "Modify the configuration content of the golang service in bulk",
	Long: `Modify the configuration content of the golang service in bulk, regardless of enabled or disabled services. 
Submitting in the format of "key=value" will automatically modify the specified items in all "*.conf" files under the specified path.`,
	Run: func(cmd *cobra.Command, args []string) {
		dir, err := cmd.Flags().GetString("dir")
		if err != nil {
			dir = getExecDir()
		}
		values, err := cmd.Flags().GetStringSlice("value")
		if err != nil {
			println(err.Error())
			return
		}
		// 列出所有文件
		cmdd := exec.Command("find", dir, "-name", `*.conf`)
		cmdd.Dir = dir
		var flst = make([]string, 0)
		if b, err := cmdd.CombinedOutput(); err != nil {
			println("search configure file failed: " + string(b))
			return
		} else {
			ss := strings.Split(string(b), "\n")
			for _, v := range ss {
				if strings.HasSuffix(v, ".conf") {
					flst = append(flst, v)
				}
			}
		}
		var result = "These files update successfully:"
		for _, v := range flst {
			path, _ := filepath.Abs(v)
			if replaceConfFile(path, values) {
				result += "\n  " + path
			}
		}
		println(result)
	},
}

func replaceConfFile(file string, values []string) bool {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return false
	}
	lines := strings.Split(string(b), "\n")
	var found = false
	var newfile bytes.Buffer
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "#" || line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			newfile.WriteString(line + "\n")
			continue
		}
		if !strings.Contains(line, "=") {
			continue
		}
		var mathcvalue = false
		for _, value := range values {
			v := strings.SplitN(value, "=", 2)
			k := strings.SplitN(line, "=", 2)
			if strings.TrimSpace(k[0]) == strings.TrimSpace(v[0]) {
				newfile.WriteString(value + "\n\n")
				mathcvalue = true
				found = true
				break
			}
		}
		if !mathcvalue {
			newfile.WriteString(line + "\n\n")
		}
	}
	if found {
		if err := ioutil.WriteFile(file, newfile.Bytes(), 0664); err != nil {
			println("update file " + file + " failed: " + err.Error())
			return false
		}
	}
	return found
}

func init() {
	rootCmd.AddCommand(configsvrCmd)

	configsvrCmd.Flags().StringP("dir", "d", ".", `Specify the path of the configuration file to be searched, the default is the current path`)
	configsvrCmd.Flags().StringSliceP("value", "v", []string{}, `Set the configuration item to be modified, "key=value" format`)
	configsvrCmd.MarkFlagRequired("value")
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// configsvrCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// configsvrCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
