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

package server

import (
	"context"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/util"
	log "github.com/sirupsen/logrus"
)

// Remove remove a server container
func Remove(srvCfg *config.Config) error {
	log.Infof("removing server container %s ...", srvCfg.Server.Name)

	cont, err := DockerCli.ContainerInspect(context.Background(), util.GetContainerName(srvCfg.Docker.NamePrefix, srvCfg.Server.Name))
	if err != nil {
		if client.IsErrNotFound(err) {
			log.Infof("server container %s doesn't exist, no removal done", srvCfg.Server.Name)
			return nil
		}
		return err
	}

	normalizedStatus := strings.ToLower(cont.State.Status)
	if normalizedStatus == "running" {
		log.Warnf("server container %s still running, can't remove it", srvCfg.Server.Name)
		return nil
	}

	if err = DockerCli.ContainerRemove(context.Background(), cont.ID, types.ContainerRemoveOptions{}); err != nil {
		return err
	}

	for i := 0; i < 10; i++ {
		cont, err = DockerCli.ContainerInspect(context.Background(), util.GetContainerName(srvCfg.Docker.NamePrefix, srvCfg.Server.Name))
		if err != nil {
			if client.IsErrNotFound(err) {
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	log.Infof("removed server container %s", srvCfg.Server.Name)
	return nil
}
