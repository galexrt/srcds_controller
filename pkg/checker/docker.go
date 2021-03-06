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

package checker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/galexrt/srcds_controller/pkg/server"
	"github.com/galexrt/srcds_controller/pkg/userconfig"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// CheckForDockerEvents check for docker container events and react to certain events
func CheckForDockerEvents(stopCh <-chan struct{}) {
	filterArgs := filters.NewArgs()
	filterArgs.Add("app", "gameserver")
	filterArgs.Add("managed-by", "srcds_controller")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	eventStream, errChan := server.DockerCli.Events(ctx, types.EventsOptions{
		Filters: filterArgs,
		Since:   "0s",
	})

	for {
		select {
		case <-stopCh:
			return
		case event := <-eventStream:
			log.Debug("received docker event")
			if _, ok := event.Actor.Attributes["name"]; ok {
				// Iterate over usercfg.Cfg.Servers and check which server matches by name
				for _, srv := range userconfig.Cfg.Servers {
					if event.Actor.Attributes["name"] == srv.Docker.NamePrefix+srv.Server.Name {
						event.Actor.Attributes["name"] = strings.TrimPrefix(event.Actor.Attributes["name"], srv.Docker.NamePrefix)
						if err := handleDockerEvent(event); err != nil {
							log.Error(err)
						}
					}
				}
			} else {
				log.Error(fmt.Errorf("no container name in docker event attributes"))
			}
		case err := <-errChan:
			if err != nil {
				log.WithError(err).Error("received error from docker events stream")
				return
			}
		}
	}
}

func handleDockerEvent(event events.Message) error {
	eventAction := strings.ToLower(event.Action)
	switch eventAction {
	case "die":
		if _, ok := event.Actor.Attributes["name"]; !ok {
			return fmt.Errorf("docker event: event has no container name in it")
		}
		serverName := event.Actor.Attributes["name"]
		serverCfg, ok := userconfig.Cfg.Servers[serverName]
		if !ok {
			return fmt.Errorf("docker event: unable to find server config for %s", serverName)
		}

		if !serverCfg.Server.Enabled {
			return nil
		}

		if viper.GetBool("dry-run") {
			log.WithField("server", serverName).Info("docker event: dry-run mode active, server restart")
		} else {
			log.WithField("server", serverName).Info("docker event: restarting server")
			if err := server.Restart(serverCfg); err != nil {
				return err
			}
		}
	default:
		log.WithField("event_action", eventAction).Debugf("docker event: event isn't of our concern (not of type 'die')")
	}
	return nil
}
