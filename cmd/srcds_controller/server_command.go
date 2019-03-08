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
	"context"

	"github.com/docker/docker/client"
	"github.com/galexrt/srcds_controller/pkg/server"
	"github.com/galexrt/srcds_controller/pkg/util"
	"github.com/spf13/cobra"
)

// serverCommandCmd represents the stop command
var serverCommandCmd = &cobra.Command{
	Use:   "command",
	Short: "'Align' server in regards to rcon password, logecho and others.",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]
		logger.Infof("running command in server %s", serverName)

		cli, err := client.NewClientWithOpts(client.FromEnv)
		if err != nil {
			return err
		}

		if _, err := cli.ContainerInspect(context.Background(), util.GetContainerName(serverName)); err != nil {
			return err
		}

		return server.SendCommand(args[0], args[1:])
	},
}

func init() {
	serverCmd.AddCommand(serverCommandCmd)
}
