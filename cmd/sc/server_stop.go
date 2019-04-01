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
	"github.com/galexrt/srcds_controller/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serverStopCmd represents the stop command
var serverStopCmd = &cobra.Command{
	Use:               "stop",
	Short:             "Stop one or more servers",
	PersistentPreRunE: initDockerCli,
	RunE: func(cmd *cobra.Command, args []string) error {
		var errs util.Errors
		var servers []string
		if viper.GetBool("all") {
			for _, srv := range config.Cfg.Servers {
				servers = append(servers, srv.Name)
			}
		} else {
			servers = viper.GetStringSlice("servers")
		}
		if len(servers) == 0 {
			return fmt.Errorf("no server list given (`--servers=SERVER_A,SERVER_B`) or `--all` flag not given (can also mean that no servers are in the config)")
		}
		wg := sync.WaitGroup{}
		for _, serverName := range servers {
			wg.Add(1)
			go func(serverName string) {
				defer wg.Done()
				if err := server.Stop(serverName); err != nil {
					errs.Lock()
					errs.Errs = append(errs.Errs, err)
					errs.Unlock()
				}
			}(serverName)
		}
		wg.Wait()
		if len(errs.Errs) > 0 {
			err := errors.New("error occured during server stop")
			for _, erro := range errs.Errs {
				err = errors.Wrap(err, erro.Error())
			}
			return err
		}
		return nil
	},
}

func init() {
	serverStopCmd.PersistentFlags().DurationP("timeout", "t", 15*time.Second, "Server stop timeout before kill will be triggered")
	viper.BindPFlag("timeout", serverStopCmd.PersistentFlags().Lookup("timeout"))

	rootCmd.AddCommand(serverStopCmd)
}
