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
	"time"

	"github.com/galexrt/srcds_controller/pkg/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serverRestartCmd represents the restart command
var serverRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart one or more servers",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := server.Stop(cmd, args); err != nil {
			return err
		}
		if viper.GetBool("remove") {
			if err := server.Remove(cmd, args); err != nil {
				return err
			}
		}
		return server.Start(cmd, args)
	},
}

func init() {
	serverCmd.AddCommand(serverRestartCmd)

	serverRestartCmd.PersistentFlags().BoolP("remove", "r", false, "Remove server container after stop")
	serverRestartCmd.PersistentFlags().DurationP("timeout", "t", 15*time.Second, "Server stop timeout before kill will be triggered")

	viper.BindPFlag("remove", serverRestartCmd.PersistentFlags().Lookup("remove"))
	viper.BindPFlag("timeout", serverRestartCmd.PersistentFlags().Lookup("timeout"))
}
