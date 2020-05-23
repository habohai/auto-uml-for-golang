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

	"github.com/haibeihabo/auto-uml-for-golang/pkg/logging"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// logCmd represents the log command
var logCmd = &cobra.Command{
	Use:   "log",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("log called")
		logging.SetUp()
	},
}

func init() {
	runtimeCmd.AddCommand(logCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// logCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// logCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	logCmd.Flags().StringP("path", "p", "./runtime/log", "the dir for save log file")
	logCmd.Flags().StringP("name", "n", "test", "the name for save log file")
	logCmd.Flags().StringP("ext", "e", "log", "the file ext for save log file")
	viper.BindPFlag("runtime.log.path", logCmd.Flags().Lookup("path"))
	viper.BindPFlag("runtime.log.ext", logCmd.Flags().Lookup("name"))
	viper.BindPFlag("runtime.log.name", logCmd.Flags().Lookup("ext"))
}
