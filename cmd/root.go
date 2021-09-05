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
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"sigs.k8s.io/yaml"

	doorman "github.com/meln5674/doorman/internal"
	public "github.com/meln5674/doorman/pkg/doorman"
)

var (
	cfgFile string
)

// TODO: make a "validate" command which validates a config file (and optionally testing connection to k8s) without changing any files or restarting nginx

var rootCmd = &cobra.Command{
	Use:   "doorman",
	Short: "Kubenetes Load Balancer Automation",
	Long:  `Doorman makes it simple to automatically create and update a Load Balancing server whenever nodes change`,
	Run: func(cmd *cobra.Command, args []string) {
		var cfg public.ConfigFile

		cfgBytes, err := ioutil.ReadFile(cfgFile)
		if err != nil {
			fmt.Printf("Failed to read config file: %v\n", err)
			os.Exit(1)
		}
		if err := yaml.Unmarshal(cfgBytes, &cfg); err != nil {
			fmt.Printf("Failed to unmarshal config file: %v\n", err)
			os.Exit(1)
		}

		// TODO: Handle SIGINT as graceful shutdown
		// TODO: Handle SIGHUP as config reload
		// TODO: Implement proper logging (klog?)
		ctx := context.Background()
		stop := make(chan struct{})
		app := doorman.Doorman{}
		fmt.Println(cfg)
		fmt.Println("Loading config...")
		if err := app.FromConfig(&cfg); err != nil {
			fmt.Printf("Failed to parse config file: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(app)
		fmt.Println("Running...")
		if err := app.Run(ctx, stop); err != nil {
			fmt.Printf("Stopping with error: %v\n", err)
			os.Exit(1)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "/etc/nginx/doorman.yaml", "Path to config file")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
