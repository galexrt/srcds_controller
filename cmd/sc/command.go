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
	"fmt"
	"sync"

	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serverCommandCmd represents the stop command
var serverCommandCmd = &cobra.Command{
	Use: "command",
	Aliases: []string{
		"cmd",
		"c",
	},
	Short:             "Send a command to one or more servers",
	Args:              cobra.MinimumNArgs(1),
	PersistentPreRunE: initDockerCli,
	RunE: func(cmd *cobra.Command, args []string) error {
		servers, err := checkServers(cmd, args)
		if err != nil {
			return err
		}

		var cmdArgs []string
		if len(args) > 1 {
			cmdArgs = args[1:]
		} else {
			cmdArgs = args[0:]
		}
		if viper.GetBool("all") {
			cmdArgs = args[0:]
		}

		errorOccured := false
		wg := sync.WaitGroup{}
		for _, serverCfg := range servers {
			wg.Add(1)
			go func(cfg *config.Config) {
				defer wg.Done()
				if err := server.SendCommand(cfg, cmdArgs); err != nil {
					log.Errorf("%+v", err)
					errorOccured = true
				}
			}(serverCfg)
		}
		wg.Wait()
		if errorOccured {
			return fmt.Errorf("error when sending commands")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serverCommandCmd)
}
