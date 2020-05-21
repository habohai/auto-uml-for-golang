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
)

// codeargsCmd represents the codeargs command
var codeargsCmd = &cobra.Command{
	Use:   "codeargs",
	Short: "code agrs",
	Long:  "code args like: code path, gopath, outputfile",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("codeargs called")
	},
}

func init() {
	rootCmd.AddCommand(codeargsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// codeargsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// codeargsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	codeargsCmd.Flags().StringP("code_path", "c", "", "the golang code path")
	codeargsCmd.Flags().StringP("output_path", "o", "./runtime/uml", "the path to output")
}
