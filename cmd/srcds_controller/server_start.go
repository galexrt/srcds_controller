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
	"github.com/galexrt/srcds_controller/pkg/server"
	"github.com/spf13/cobra"
)

// serverStartCmd represents the start command
var serverStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start one or more servers",
	Args:  cobra.MinimumNArgs(1),
	RunE:  server.Start,
}

func init() {
	serverCmd.AddCommand(serverStartCmd)
}