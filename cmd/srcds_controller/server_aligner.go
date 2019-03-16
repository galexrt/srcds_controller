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

	"github.com/docker/docker/client"
	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/server"
	"github.com/galexrt/srcds_controller/pkg/util"
	"github.com/spf13/cobra"
)

// serverAlignerCmd represents the aligner command
var serverAlignerCmd = &cobra.Command{
	Use:               "aligner",
	Short:             "'Align' server in regards to rcon password, logecho and others.",
	Hidden:            true,
	PersistentPreRunE: initDockerCli,
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, serverCfg := range config.Cfg.Servers {
			logger.Infof("aligning server %s ...", serverCfg.Name)

			if !serverCfg.Enabled {
				logger.Infof("skipping server alignment for %s as is disabled", serverCfg.Name)
				continue
			}

			// Check if server container is running, if not start, unless config says server disabled.
			cont, err := server.GetServerContainer(util.GetContainerName(serverCfg.Name))
			if err != nil && !client.IsErrNotFound(err) {
				logger.Errorf("failed to align server %s, during get server container. %+v", serverCfg.Name, err)
				continue
			}
			if client.IsErrNotFound(err) || !cont.State.Running {
				if err := server.Start(serverCfg.Name); err != nil {
					logger.Errorf("failed to align server %s, during try to start not running server. %+v", serverCfg.Name, err)
					continue
				}
				// TODO Send command to trigger "I AM READY!" message then
				// use server.WaitForConsoleContains() function.
				//
				// Right now we just sleep 32 seconds and continue.
				found, err := server.WaitForConsoleContains(serverCfg.Name, "I AM READY!")
				if err != nil {
					logger.Errorf("failed to align server %s, during wait for console to contain ready signal. %+v", serverCfg.Name, err)
					continue
				}
				if !found {
					logger.Errorf("failed to align server %s, during wait for console to contain ready signal. did not find start up done signal text", serverCfg.Name)
					continue
				}
				time.Sleep(32 * time.Second)
			}

			//server.SendCommand(serverCfg.Name, "sv_logecho 1")
			if err := server.UpdateRCONPassword(serverCfg.Name, serverCfg.RCON.Password); err != nil {
				logger.Errorf("failed to align server %s. %+v", serverCfg.Name, err)
				continue
			}
			logger.Infof("aligned server %s.", serverCfg.Name)
		}

		return nil
	},
}

func init() {
	serverCmd.AddCommand(serverAlignerCmd)
}
