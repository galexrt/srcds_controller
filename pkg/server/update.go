/*
Copyright 2021 Alexander Trost <galexrt@googlemail.com>

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
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/galexrt/srcds_controller/pkg/config"
	log "github.com/sirupsen/logrus"
)

// SteamCMDUpdate runs steamcmd.sh for a server
func SteamCMDUpdate(serverCfg *config.Config, beta string) error {
	// Set the server dir as the home, unless otherwise set
	serverHomeDir := serverCfg.Server.Path
	if serverCfg.Server.RunOptions.HomeDir != "" {
		serverHomeDir = serverCfg.Server.RunOptions.HomeDir
	}
	serverPath := path.Clean(serverCfg.Server.Path)

	logger := log.WithFields(log.Fields{
		"server": serverCfg.Server.Name,
		"path":   serverPath,
	})

	argAppUpdate := fmt.Sprintf("+app_update %d", serverCfg.Server.GameID)
	if beta != "" {
		argAppUpdate += fmt.Sprintf(" -beta %s", beta)
	}
	argAppUpdate += " validate"

	containerName := fmt.Sprintf("steamcmd_update-%s", serverCfg.Server.Name)

	log.Debug("check if container exists and remove")
	if _, err := DockerCli.ContainerInspect(context.Background(), containerName); err != nil {
		if !client.IsErrNotFound(err) {
			return err
		}
	} else {
		// Container exists so remove it
		log.Info("container exists, removing it")
		if err := DockerCli.ContainerRemove(context.Background(), containerName, types.ContainerRemoveOptions{}); err != nil {
			return err
		}
	}

	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	steamCMDDir := serverCfg.Server.SteamCMDDir
	if steamCMDDir == "" {
		steamCMDDir = path.Join(serverHomeDir, "steamcmd")
	}
	log.Debugf("server dir: %s, steamcmd dir: %+s", serverPath, steamCMDDir)

	contCfg := &container.Config{
		Labels: map[string]string{
			"app":        "steamcmd",
			"managed-by": "srcds_controller",
		},
		Env: []string{
			fmt.Sprintf("HOME=%s", serverHomeDir),
		},
		AttachStdin: false,
		Tty:         true,
		OpenStdin:   false,
		Hostname:    hostname,
		User:        fmt.Sprintf("%d:%d", serverCfg.Server.RunOptions.UID, serverCfg.Server.RunOptions.GID),
		Image:       *serverCfg.Docker.Image,
		WorkingDir:  serverPath,
		Entrypoint: strslice.StrSlice{
			path.Join(steamCMDDir, "steamcmd.sh"),
		},
		Cmd: strslice.StrSlice{
			"+login anonymous",
			fmt.Sprintf("+force_install_dir %s", serverPath),
			argAppUpdate,
			"+quit",
		},
	}

	contHostCfg := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   serverCfg.Docker.LocalTimeFile,
				Target:   "/etc/localtime",
				ReadOnly: true,
			},
			{
				Type:     mount.TypeBind,
				Source:   serverCfg.Docker.TimezoneFile,
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
				Source:   serverPath,
				Target:   serverPath,
				ReadOnly: false,
			},
			// SteamCMD directory
			{
				Type:     mount.TypeBind,
				Source:   steamCMDDir,
				Target:   steamCMDDir,
				ReadOnly: false,
			},
		},
		NetworkMode: "host",
		AutoRemove:  true,
	}

	netCfg := &network.NetworkingConfig{}
	var resp container.ContainerCreateCreatedBody
	resp, err = DockerCli.ContainerCreate(context.Background(), contCfg, contHostCfg, netCfg, containerName)
	if err != nil {
		return err
	}

	for _, warning := range resp.Warnings {
		log.Warning(warning)
	}

	logger.Infof("running steamcmd.sh with args: %s", contCfg.Cmd)

	if err = DockerCli.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start steamcmd container. %+v", err)
	}
	defer func() {
		stopTimeout := 5 * time.Second
		if err := DockerCli.ContainerStop(context.Background(), resp.ID, &stopTimeout); err != nil {
			logger.Errorf("unable to stop steamcmd container. %+v", err)
		}
	}()

	logsContext := context.Background()
	logStream, err := DockerCli.ContainerLogs(logsContext, resp.ID, types.ContainerLogsOptions{
		Follow:     true,
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(logStream)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		if err != io.EOF {
			return err
		}
	}

	return nil
}
