/*
Copyright 2020 Alexander Trost <galexrt@googlemail.com>

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

	"github.com/docker/docker/client"
	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serverToolsRestartEmpty represents the stop command
var serverToolsRestartEmpty = &cobra.Command{
	Hidden:            true,
	Use:               "restartempty",
	Short:             "Use quit_nice command, wait till server exits / quits and restart server when exited.",
	PersistentPreRunE: initDockerCli,
	RunE: func(cmd *cobra.Command, args []string) error {
		servers, err := checkServers(cmd, args)
		if err != nil {
			return err
		}

		for _, srv := range servers {
			if err := server.SendCommand(srv, []string{
				"quit_nice",
			}); err != nil {
				return err
			}
		}

		waitTime := viper.GetDuration("wait-time")

		secsTotal := int(waitTime.Seconds())
		secsRemaining := secsTotal

		timeLoggerCoolDown := 15
		for {
			wg := sync.WaitGroup{}
			for i, srv := range servers {
				stopped := false
				cont, err := server.GetServerContainer(srv)
				if err != nil {
					if !client.IsErrNotFound(err) {
						log.Errorf("error getting server %s container. %+v", srv.Server.Name, err)
						continue
					}
					stopped = true
				}

				if cont.ContainerJSONBase != nil {
					normalizedStatus := strings.ToLower(cont.State.Status)
					if normalizedStatus != "running" {
						stopped = true
					}
				}

				if stopped {
					log.Infof("server %s is not running anymore, starting up again", srv.Server.Name)

					wg.Add(1)
					go func(cfg *config.Config) {
						defer wg.Done()

						if err := server.Stop(cfg); err != nil {
							log.Errorf("error during server container stop. %+v", err)
						}

						if viper.GetBool("remove") {
							if err := server.Remove(cfg); err != nil {
								log.Errorf("error during server container removal. %+v", err)
							}
						}

						if !viper.GetBool("stop-only") {
							time.Sleep(500 * time.Millisecond)
							if err := server.Start(cfg); err != nil {
								log.Errorf("error during server start. %+v", err)
							}
						}
					}(srv)

					// Remove server from list
					servers[i] = servers[len(servers)-1]
					servers = servers[:len(servers)-1]

					log.Infof("restart when empty completed for server %s", srv.Server.Name)
				}
			}

			wg.Wait()

			if secsRemaining <= 0 {
				return fmt.Errorf("timed out waiting for server(s) to quit after %s", waitTime)
			}

			if timeLoggerCoolDown == 15 || timeLoggerCoolDown == 0 {
				log.Infof("wait time: remaining: %d seconds, total: %d seconds", secsRemaining, secsTotal)
				timeLoggerCoolDown = 15
			}
			timeLoggerCoolDown--

			time.Sleep(1 * time.Second)
			secsRemaining--
		}
	},
}

func init() {
	serverToolsRestartEmpty.PersistentFlags().DurationP("wait-time", "w", 15*time.Minute, "Time to wait for server container to exit")
	serverToolsRestartEmpty.PersistentFlags().Bool("stop-only", false, "If servers should only be stopped and not restarted")
	serverToolsRestartEmpty.PersistentFlags().BoolP("remove", "r", true, "Remove the server container on restart")
	viper.BindPFlag("wait-time", serverToolsRestartEmpty.PersistentFlags().Lookup("wait-time"))
	viper.BindPFlag("stop-only", serverToolsRestartEmpty.PersistentFlags().Lookup("stop-only"))
	viper.BindPFlag("remove", serverToolsRestartEmpty.PersistentFlags().Lookup("remove"))

	serverToolsCmd.AddCommand(serverToolsRestartEmpty)
}
