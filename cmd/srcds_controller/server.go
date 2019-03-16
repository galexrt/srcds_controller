/*
Copyright 2019 Alexander Trost <galexrt@googlemail.com>

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
	"github.com/docker/docker/client"
	"github.com/galexrt/srcds_controller/pkg/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "server control/management subcommand section",
}

func init() {
	serverCmd.PersistentFlags().StringSlice("servers", []string{}, "Comma separated list of servers")
	serverCmd.PersistentFlags().Bool("all", false, "If all servers should be used")
	viper.BindPFlag("servers", serverCmd.PersistentFlags().Lookup("servers"))
	viper.BindPFlag("all", serverCmd.PersistentFlags().Lookup("all"))

	rootCmd.AddCommand(serverCmd)
}

func initDockerCli(cmd *cobra.Command, args []string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}
	server.DockerCli = cli
	return err
}
