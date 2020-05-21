/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

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
	"github.com/spf13/viper"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of auto-uml-for-golang",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("-------------------")
		fmt.Println(viper.Get("runtime.log.savepath"))
		fmt.Println(viper.Get("runtime.log.fileext"))
		fmt.Println(viper.Get("runtime.log.savename"))
		fmt.Println(viper.Get("codeargs.codepath"))
		fmt.Println(viper.Get("codeargs.outputpath"))
		fmt.Println(viper.Get("config"))
		fmt.Println(viper.Get("goenv.gopath"))
		fmt.Println("-------------------")

		fmt.Println("cobratest version is v0.0.1")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
