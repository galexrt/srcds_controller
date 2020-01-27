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
	"fmt"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-units"
	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/util"
	log "github.com/sirupsen/logrus"
)

// Start starts a server
func Start(serverCfg *config.Config) error {
	log.Infof("starting server %s ...", serverCfg.Server.Name)

	cont, err := DockerCli.ContainerInspect(context.Background(), util.GetContainerName(serverCfg.Docker.NamePrefix, serverCfg.Server.Name))
	if err != nil && !client.IsErrNotFound(err) {
		return err
	} else if err == nil && (cont.State.Status != "created" && cont.State.Status != "exited") {
		return fmt.Errorf("server %s container is already existing / running", serverCfg.Server.Name)
	}

	var containerID string
	if cont.ContainerJSONBase != nil && (cont.State.Status == "created" || cont.State.Status == "exited") {
		containerID = cont.ID
	} else {
		serverDir := serverCfg.Server.Path
		mountDir := serverCfg.Server.MountsDir

		var hostname string
		hostname, err = os.Hostname()
		if err != nil {
			return err
		}

		contCfg := &container.Config{
			Labels: map[string]string{
				"app":        "gameserver",
				"managed-by": "srcds_controller",
			},
			Env:         []string{},
			AttachStdin: true,
			Tty:         false,
			OpenStdin:   true,
			Hostname:    hostname,
			User:        fmt.Sprintf("%d:%d", serverCfg.Server.RunOptions.UID, serverCfg.Server.RunOptions.GID),
			Image:       *serverCfg.Docker.Image,
			WorkingDir:  serverDir,
		}

		contHostCfg := &container.HostConfig{
			RestartPolicy: container.RestartPolicy{
				Name: "no",
			},
			Mounts: []mount.Mount{
				{
					Type:     mount.TypeBind,
					Source:   "/etc/localtime",
					Target:   "/etc/localtime",
					ReadOnly: true,
				},
				{
					Type:     mount.TypeBind,
					Source:   "/etc/timezone",
					Target:   "/etc/timezone",
					ReadOnly: true,
				},
				{
					Type:     mount.TypeBind,
					Source:   "/etc/passwd",
					Target:   "/etc/passwd",
					ReadOnly: true,
				},
				{
					Type:     mount.TypeBind,
					Source:   "/etc/group",
					Target:   "/etc/group",
					ReadOnly: true,
				},
				// Server directory
				{
					Type:     mount.TypeBind,
					Source:   serverDir,
					Target:   serverDir,
					ReadOnly: false,
				},
			},
			NetworkMode: "host",
		}
		if mountDir != "" {
			contHostCfg.Mounts = append(contHostCfg.Mounts, mount.Mount{
				Type:     mount.TypeBind,
				Source:   mountDir,
				Target:   mountDir,
				ReadOnly: true,
			})
		}
		if serverCfg.Server.Resources != nil {
			contHostCfg.Resources = *serverCfg.Server.Resources
		}

		// Disable Core dumps for the containers. GMod and other games seem to
		// do core dumps for random reasons but we don't need them
		contHostCfg.Ulimits = []*units.Ulimit{
			&units.Ulimit{
				Name: "core",
				Hard: 0,
			},
		}

		netCfg := &network.NetworkingConfig{}
		var resp container.ContainerCreateCreatedBody
		resp, err = DockerCli.ContainerCreate(context.Background(), contCfg, contHostCfg, netCfg, util.GetContainerName(serverCfg.Docker.NamePrefix, serverCfg.Server.Name))
		if err != nil {
			return err
		}

		for _, warning := range resp.Warnings {
			log.Warning(warning)
		}
		containerID = resp.ID
	}

	if err = DockerCli.ContainerStart(context.Background(), containerID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	log.Infof("started server %s", serverCfg.Server.Name)

	return nil
}
