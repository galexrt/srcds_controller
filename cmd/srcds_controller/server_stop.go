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

// serverStopCmd represents the stop command
var serverStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop one or more servers",
	Args:  cobra.MinimumNArgs(1),
	RunE:  server.Stop,
}

func init() {
	serverCmd.AddCommand(serverStopCmd)

	serverStopCmd.PersistentFlags().DurationP("timeout", "t", 15*time.Second, "Server stop timeout before kill will be triggered")

	viper.BindPFlag("timeout", serverStopCmd.PersistentFlags().Lookup("timeout"))
}
