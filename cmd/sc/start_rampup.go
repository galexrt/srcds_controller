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
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serverStartRampUpCmd represents the start command
var serverStartRampUpCmd = &cobra.Command{
	Use:               "rampup",
	Short:             "Ramp up servers one by one with delay between",
	PersistentPreRunE: initDockerCli,
	RunE: func(cmd *cobra.Command, args []string) error {
		if !viper.IsSet("remove") {
			viper.Set("remove", true)
		}

		servers, err := checkServers(cmd, args)
		if err != nil {
			return err
		}

		for k, cfg := range servers {
			if viper.GetBool("remove") {
				if err := server.Remove(cfg); err != nil {
					log.Errorf("error removing server. %+v", err)
				}
			}

			if err := server.Start(cfg); err != nil {
				log.Errorf("error during server %s start. %+v", cfg.Server.Name, err)
			}

			if k+1 != len(servers) {
				time.Sleep(viper.GetDuration("delay"))
			}
		}
		return nil
	},
}

func init() {
	serverStartRampUpCmd.PersistentFlags().DurationP("delay", "d", 30*time.Second, "Delay between each server start")
	viper.BindPFlag("delay", serverStartRampUpCmd.PersistentFlags().Lookup("delay"))

	serverStartCmd.AddCommand(serverStartRampUpCmd)
}
