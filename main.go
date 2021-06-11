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
package main

import (
	"fmt"

	cmd "./cmd"
)

var (
	version     = "0.0.0"
	goVersion   = ""
	buildDate   = ""
	platform    = ""
	author      = "X.Yuan"
	programName = "Background process controller"
)

func main() {
	cmd.SetVersion(
		fmt.Sprintf("\n%s\r\nVersion:\t%s\r\nGo version:\t%s\r\nBuild date:\t%s\r\nBuild OS:\t%s\r\nCode by:\t%s\r\n",
			programName,
			version,
			goVersion,
			platform,
			buildDate,
			author))
	cmd.Execute()
}
