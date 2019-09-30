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
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

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
			if _, ok := event.Actor.Attributes["name"]; ok {
				event.Actor.Attributes["name"] = strings.TrimPrefix(event.Actor.Attributes["name"], config.Cfg.Docker.NamePrefix)
			} else {
				log.Error(fmt.Errorf("no container name in docker event attributes"))
				break
			}
			if err := handleDockerEvent(event); err != nil {
				log.Error(err)
			}
		case err := <-errChan:
			if err != nil {
				log.Error(err)
				return
			}
		}
	}
}

func handleDockerEvent(event events.Message) error {
	switch strings.ToLower(event.Action) {
	case "die":
		if _, ok := event.Actor.Attributes["name"]; !ok {
			return fmt.Errorf("given event has no container name in it")
		}
		serverName := event.Actor.Attributes["name"]

		if viper.GetBool("dry-run") {
			log.Debug("dry-run mode active, server restart")
		} else {
			time.Sleep(5 * time.Second)
			if err := server.Restart(serverName); err != nil {
				return err
			}
		}
	default:
		log.Debugf("docker event that isn't of our concern (not 'die')")
	}
	return nil
}
