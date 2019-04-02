// Copyright © 2019 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"os"

	"github.com/lentil1016/descheduler/pkg/config"
	"github.com/spf13/cobra"
)

var configFile string
var kubeConfigFile string
var dryRun bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "rescheduler [FLAGS]",
	Short: "Kubernetes pod rescheduler.",
	Long:  "This tool deschedule proper pod in kubernetes cluster in order to reschedule it.",
	Args:  cobra.MaximumNArgs(0),
	Run:   doDescheduleCmd,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "deschedule policy config file (default is $HOME/.kubewatch.yaml)")
	rootCmd.PersistentFlags().StringVarP(&kubeConfigFile, "kubeconfig", "k", "", "kubeConfig file (default is $HOME/.kube/config)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "use dry run mod")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	config.InitConfig(configFile, kubeConfigFile, dryRun)
}
