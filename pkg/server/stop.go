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

	"github.com/docker/docker/client"
	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Stop stops a server
func Stop(serverCfg *config.Config) error {
	log.Infof("stopping server %s ...", serverCfg.Server.Name)

	cont, err := DockerCli.ContainerInspect(context.Background(), util.GetContainerName(serverCfg.Docker.NamePrefix, serverCfg.Server.Name))
	if err != nil {
		if client.IsErrNotFound(err) {
			return nil
		}
		return err
	}

	if cont.State.Running {
		duration := viper.GetDuration("timeout")
		if err = DockerCli.ContainerStop(context.Background(), cont.ID, &duration); err != nil {
			return err
		}
	}

	log.Infof("stopped server %s (container: %s)", serverCfg.Server.Name, cont.ID)
	return nil
}
