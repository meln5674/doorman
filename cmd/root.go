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
	"os"

	"github.com/spf13/viper"

	doorman "github.com/meln5674/doorman/internal"
	public "github.com/meln5674/doorman/pkg/doorman"
)

var (
	cfgFile string
	cfg     public.ConfigFile
)

var rootCmd = &cobra.Command{
	Use:   "doorman",
	Short: "Kubenetes Load Balancer Automation",
	Long:  `Doorman makes it simple to automatically create and update a Load Balancing server whenever nodes change`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := viper.Unmarshal(&cfg); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// TODO: Handle SIGINT as graceful shutdown
		// TODO: Handle SIGHUP as config reload
		// TODO: Implement proper logging (klog?)
		ctx := context.Background()
		stop := make(chan struct{})
		app := doorman.Doorman{}
		if err := app.FromConfig(&cfg); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if err := app.Run(ctx, stop); err != nil {
			fmt.Println(err)
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
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/doorman.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name "doorman" (without extension).
		viper.AddConfigPath(home)
		viper.AddConfigPath("/etc/nginx/doorman.yaml")
		viper.SetConfigType("yaml")
		viper.SetConfigName("doorman")
	}

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}

}
