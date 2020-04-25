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
	"time"

	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serverStopCmd represents the stop command
var serverStopCmd = &cobra.Command{
	Use:               "stop",
	Short:             "Stop one or more servers",
	PersistentPreRunE: initDockerCli,
	RunE: func(cmd *cobra.Command, args []string) error {
		servers, err := checkServers(cmd, args)
		if err != nil {
			return err
		}

		errorOccured := false
		wg := sync.WaitGroup{}
		for _, serverCfg := range servers {
			wg.Add(1)
			go func(cfg *config.Config) {
				defer wg.Done()

				if err := server.Stop(cfg); err != nil {
					log.Errorf("error during container stop. %+v", err)
					errorOccured = true
				}

				if err := server.Remove(cfg); err != nil {
					log.Errorf("error during container removal. %+v", err)
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
	serverStopCmd.PersistentFlags().DurationP("timeout", "t", 15*time.Second, "Server stop timeout before kill will be triggered")
	serverStopCmd.PersistentFlags().BoolP("remove", "r", false, "Remove the server container before starting if it exists")
	viper.BindPFlag("timeout", serverStopCmd.PersistentFlags().Lookup("timeout"))
	viper.BindPFlag("remove", serverStopCmd.PersistentFlags().Lookup("remove"))

	rootCmd.AddCommand(serverStopCmd)
}
