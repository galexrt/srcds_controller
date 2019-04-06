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

	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/server"
	"github.com/galexrt/srcds_controller/pkg/util"
	"github.com/pkg/errors"
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
	Args:              cobra.MinimumNArgs(2),
	PersistentPreRunE: initDockerCli,
	RunE: func(cmd *cobra.Command, args []string) error {
		var errs util.Errors
		var servers []string
		if viper.GetBool(AllServers) || strings.ToLower(args[0]) == AllServers {
			for _, srv := range config.Cfg.Servers {
				servers = append(servers, srv.Name)
			}
		} else {
			servers = strings.Split(args[0], ",")
		}
		if len(servers) == 0 {
			return fmt.Errorf("no server(s) given, please put a server list as the first argument, example: `sc " + cmd.Name() + " SERVER_A,SERVER_B` or `all` instead of the server list")
		}
		wg := sync.WaitGroup{}
		for _, serverName := range servers {
			wg.Add(1)
			go func(serverName string) {
				defer wg.Done()
				if err := server.SendCommand(serverName, args[1:]); err != nil {
					errs.Lock()
					errs.Errs = append(errs.Errs, err)
					errs.Unlock()
				}
			}(serverName)
		}
		wg.Wait()
		if len(errs.Errs) > 0 {
			err := errors.New("error occured during command sending")
			for _, erro := range errs.Errs {
				err = errors.Wrap(err, erro.Error())
			}
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serverCommandCmd)
}
