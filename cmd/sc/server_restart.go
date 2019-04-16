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
	"strings"
	"sync"
	"time"

	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"istio.io/istio/pkg/log"
)

// serverRestartCmd represents the restart command
var serverRestartCmd = &cobra.Command{
	Use:               "restart",
	Short:             "Restart one or more servers",
	PersistentPreRunE: initDockerCli,
	RunE: func(cmd *cobra.Command, args []string) error {
		var servers []string
		if viper.GetBool(AllServers) || strings.ToLower(args[0]) == AllServers {
			for _, srv := range config.Cfg.Servers {
				servers = append(servers, srv.Name)
			}
		} else {
			servers = strings.Split(args[0], ",")
		}
		if len(servers) == 0 {
			return fmt.Errorf("no server(s) given, please provide a server list as the first argument, example: `sc " + cmd.Name() + " SERVER_A,SERVER_B` or `all` instead of the server list")
		}
		errorOccured := false
		wg := sync.WaitGroup{}
		for _, serverName := range servers {
			wg.Add(1)
			go func(serverName string) {
				defer wg.Done()
				if err := server.Restart(serverName); err != nil {
					log.Errorf("%+v", err)
					errorOccured = true
				}
			}(serverName)
		}
		wg.Wait()
		if errorOccured {
			return fmt.Errorf("error when sending commands")
		}
		return nil
	},
}

func init() {
	serverRestartCmd.PersistentFlags().DurationP("timeout", "t", 15*time.Second, "Server stop timeout before kill will be triggered")
	viper.BindPFlag("timeout", serverRestartCmd.PersistentFlags().Lookup("timeout"))

	rootCmd.AddCommand(serverRestartCmd)
}
