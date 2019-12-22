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

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/util"
)

var (
	// DockerCli Docker client
	DockerCli *client.Client
)

// GetServerContainer return container for given server name
func GetServerContainer(serverCfg *config.Config) (types.ContainerJSON, error) {
	var err error
	var cont types.ContainerJSON

	cont, err = DockerCli.ContainerInspect(context.Background(), util.GetContainerName(serverCfg.Docker.NamePrefix, serverCfg.Server.Name))
	if err != nil {
		return cont, err
	}

	return cont, nil
}
