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

	"github.com/galexrt/srcds_controller/pkg/server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serverToolsUpdate represents the stop command
var serverToolsUpdate = &cobra.Command{
	Use:               "update",
	Short:             "Update one ore more servers",
	PersistentPreRunE: initDockerCli,
	RunE: func(cmd *cobra.Command, args []string) error {
		servers, err := checkServers(cmd, args)
		if err != nil {
			return err
		}

		errorOccured := false
		for _, serverCfg := range servers {
			if err := server.SteamCMDUpdate(serverCfg, viper.GetString("beta")); err != nil {
				log.Error(err)
				errorOccured = true
			}
		}

		if errorOccured {
			return fmt.Errorf("error when sending commands")
		}
		return nil
	},
}

func init() {
	serverToolsUpdate.PersistentFlags().String("beta", "", "which branch to install during steamcmd app_install validate")
	viper.BindPFlag("beta", serverToolsUpdate.PersistentFlags().Lookup("beta"))

	serverToolsCmd.AddCommand(serverToolsUpdate)
}
