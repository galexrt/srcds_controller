/*
Copyright 2018 Alexander Trost <galexrt@googlemail.com>

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
	"time"

	"github.com/spf13/cobra"
)

// serverRestartCmd represents the restart command
var serverRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart one or more servers",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("restart called")
	},
}

func init() {
	serverCmd.AddCommand(serverRestartCmd)

	serverRestartCmd.Flags().BoolP("force", "f", false, "Force server restart")
	serverRestartCmd.Flags().DurationP("timeout", "t", 15*time.Second, "Server stop timeout before kill will be triggered")
}
