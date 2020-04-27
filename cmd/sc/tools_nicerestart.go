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
	"strconv"
	"sync"
	"time"

	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// AnnounceEveryMinute const for the announce every minute value
	AnnounceEveryMinute = "EVERY_MINUTE"
	// AnnounceEverySecond const for the announce every second value
	AnnounceEverySecond = "EVERY_SECOND"
)

// serverToolsNiceRestart represents the stop command
var serverToolsNiceRestart = &cobra.Command{
	Use:               "nicerestart",
	Short:             "Triggers a nice restart with a countdown before doing so.",
	PersistentPreRunE: initDockerCli,
	RunE: func(cmd *cobra.Command, args []string) error {
		servers, err := checkServers(cmd, args)
		if err != nil {
			return err
		}

		duration := viper.GetDuration("duration")

		errorOccured := false

		rawAnnounceTimes := viper.GetStringSlice("default-announce-times")
		for _, a := range viper.GetStringSlice("additional-announce-times") {
			rawAnnounceTimes = append(rawAnnounceTimes, a)
		}

		secsTotal := int(duration.Seconds())
		secsRemaining := secsTotal

		byMinuteAnnouncement := false
		bySecondsAnnouncement := false
		var announceTimes []string
		for _, value := range rawAnnounceTimes {
			if value == AnnounceEveryMinute {
				byMinuteAnnouncement = true
			} else if value == AnnounceEverySecond {
				bySecondsAnnouncement = true
			}
			if value != "" {
				announceTimes = append(announceTimes, value)
			}
		}

		contains := func(s []string, e string) bool {
			for _, a := range s {
				if a == e {
					return true
				}
			}
			return false
		}

		timeLoggerCoolDown := 15
	timeLoop:
		for {
			if secsRemaining <= 0 {
				wg := sync.WaitGroup{}
				for _, serverCfg := range servers {
					wg.Add(1)
					go func(cfg *config.Config) {
						defer wg.Done()

						if err := server.Stop(cfg); err != nil {
							log.Errorf("error during server stop. %+v", err)
							errorOccured = true
						}

						if viper.GetBool("remove") {
							if err := server.Remove(cfg); err != nil {
								log.Errorf("error during server container removal. %+v", err)
								errorOccured = true
							}
						}

						if !viper.GetBool("stop-only") {
							time.Sleep(500 * time.Millisecond)
							if err := server.Start(cfg); err != nil {
								log.Errorf("error during server start. %+v", err)
								errorOccured = true
							}
						}
					}(serverCfg)
				}
				wg.Wait()
				break timeLoop
			}

			var mins float64
			mins = float64(secsRemaining) / float64(60)
			if byMinuteAnnouncement && mins == float64(int64(mins)) {
				log.Info("countdown: another minute is over")
				command := fmt.Sprintf(viper.GetString("announce-minutes"), int64(mins))
				sendCommandInParallel(servers, command)
			} else if contains(announceTimes, strconv.Itoa(secsRemaining)) {
				log.Debug("countdown: need to announce")
				command := fmt.Sprintf(viper.GetString("announce-seconds"), int64(secsRemaining))
				sendCommandInParallel(servers, command)
			} else if bySecondsAnnouncement {
				command := fmt.Sprintf(viper.GetString("announce-seconds"), int64(secsRemaining))
				sendCommandInParallel(servers, command)
			}
			if timeLoggerCoolDown == 15 || timeLoggerCoolDown == 0 {
				log.Infof("countdown: remaining: %d seconds, total: %d seconds", secsRemaining, secsTotal)
				timeLoggerCoolDown = 15
			}
			timeLoggerCoolDown--

			time.Sleep(1 * time.Second)
			secsRemaining--
		}

		if errorOccured {
			return fmt.Errorf("error when sending commands")
		}
		return nil
	},
}

func init() {
	serverToolsNiceRestart.PersistentFlags().DurationP("duration", "d", 11*time.Minute, "Time to countdown for server restart")
	serverToolsNiceRestart.PersistentFlags().Bool("stop-only", false, "If servers should only be stopped and not restarted")
	serverToolsNiceRestart.PersistentFlags().String("announce-minutes", "say Server Restart in %d minute(s)!", "Command template to be sent to servers during minutes over countdown")
	serverToolsNiceRestart.PersistentFlags().String("announce-seconds", "say Server Restart in %d second(s)!", "Command template to be sent to servers during seconds over countdown")
	serverToolsNiceRestart.PersistentFlags().StringSlice("default-announce-times", []string{"EVERY_MINUTE", "45", "30", "15", "10", "9", "8", "7", "6", "5", "4", "3", "2", "1"}, "Default times  at which the left time should be announced")
	serverToolsNiceRestart.PersistentFlags().StringSlice("additional-announce-times", []string{}, "At which additional times the left time should be announced")
	serverToolsNiceRestart.PersistentFlags().BoolP("remove", "r", true, "Remove the server container on restart")
	viper.BindPFlag("duration", serverToolsNiceRestart.PersistentFlags().Lookup("duration"))
	viper.BindPFlag("stop-only", serverToolsNiceRestart.PersistentFlags().Lookup("stop-only"))
	viper.BindPFlag("announce-minutes", serverToolsNiceRestart.PersistentFlags().Lookup("announce-minutes"))
	viper.BindPFlag("announce-seconds", serverToolsNiceRestart.PersistentFlags().Lookup("announce-seconds"))
	viper.BindPFlag("default-announce-times", serverToolsNiceRestart.PersistentFlags().Lookup("default-announce-times"))
	viper.BindPFlag("additional-announce-times", serverToolsNiceRestart.PersistentFlags().Lookup("additional-announce-times"))
	viper.BindPFlag("remove", serverToolsNiceRestart.PersistentFlags().Lookup("remove"))

	serverToolsCmd.AddCommand(serverToolsNiceRestart)
}

func sendCommandInParallel(servers []*config.Config, command string) bool {
	errorOccured := false
	wg := sync.WaitGroup{}
	for _, serverCfg := range servers {
		wg.Add(1)
		go func(cfg *config.Config) {
			defer wg.Done()
			if err := server.SendCommand(cfg, []string{command}); err != nil {
				log.Errorf("error while sending command to server %s. %+v", cfg.Server.Name, err)
				errorOccured = true
			}
		}(serverCfg)
	}
	wg.Wait()
	return errorOccured
}
