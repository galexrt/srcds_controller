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
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/coreos/pkg/capnslog"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var logger = capnslog.NewPackageLogger("github.com/galexrt/srcds_controller", "server")

func Start(cmd *cobra.Command, args []string) error {
	serverName := args[0]
	logger.Infof("starting server %s\n", serverName)

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}

	cont, err := cli.ContainerInspect(context.Background(), util.GetContainerName(serverName))
	if err != nil && !client.IsErrNotFound(err) {
		return err
	} else if err == nil && (cont.State.Status != "created" && cont.State.Status != "exited") {
		return fmt.Errorf("server %s container is already existing / running", serverName)
	}

	var containerID string
	if cont.ContainerJSONBase != nil && (cont.State.Status == "created" || cont.State.Status == "exited") {
		containerID = cont.ID
	} else {
		index, serverCfg := config.Cfg.Servers.GetByName(serverName)
		if serverCfg == nil {
			return fmt.Errorf("no server config found for %s", serverName)
		}
		serverDir := serverCfg.Path

		var hostname string
		hostname, err = os.Hostname()
		if err != nil {
			return err
		}

		contArgs := strslice.StrSlice{
			"./srcds_run",
			"-port",
			strconv.Itoa(serverCfg.Port),
		}

		for _, arg := range serverCfg.Flags {
			contArgs = append(contArgs, arg)
		}

		contCfg := &container.Config{
			Env: []string{
				fmt.Sprintf("SRCDS_RUNNER_ID=%d", index),
				fmt.Sprintf("SRCDS_RUNNER_AUTH_KEY=%s", serverCfg.RCON.Password),
			},
			Cmd:         contArgs,
			AttachStdin: true,
			Tty:         false,
			OpenStdin:   true,
			Hostname:    hostname,
			User:        fmt.Sprintf("%d:%d", serverCfg.RunOptions.UID, serverCfg.RunOptions.GID),
			Image:       config.Cfg.Docker.Image,
			WorkingDir:  serverCfg.Path,
		}
		contHostCfg := &container.HostConfig{
			RestartPolicy: container.RestartPolicy{
				Name:              "on-failure",
				MaximumRetryCount: 3,
			},
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: serverDir,
					Target: serverDir,
				},
			},
			NetworkMode: "host",
		}
		netCfg := &network.NetworkingConfig{}
		resp, err := cli.ContainerCreate(context.Background(), contCfg, contHostCfg, netCfg, util.GetContainerName(serverName))
		if err != nil {
			return err
		}

		for _, warning := range resp.Warnings {
			logger.Warning(warning)
		}
		containerID = resp.ID
	}

	if err = cli.ContainerStart(context.Background(), containerID, types.ContainerStartOptions{}); err != nil {
		return err
	}

	logger.Infof("started server %s", serverName)

	return nil
}

func Stop(cmd *cobra.Command, args []string) error {
	serverName := args[0]
	logger.Infof("stopping server %s", serverName)

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}

	cont, err := cli.ContainerInspect(context.Background(), util.GetContainerName(serverName))
	if err != nil {
		if client.IsErrNotFound(err) {
			return nil
		}
		return err
	}

	containerID := cont.ID

	duration := viper.GetDuration("timeout")
	return cli.ContainerStop(context.Background(), containerID, &duration)
}

func Remove(cmd *cobra.Command, args []string) error {
	serverName := args[0]
	logger.Infof("removing server container %s", serverName)

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}

	cont, err := cli.ContainerInspect(context.Background(), util.GetContainerName(serverName))
	if err != nil {
		return err
	}

	return cli.ContainerRemove(context.Background(), cont.ID, types.ContainerRemoveOptions{})
}

func Logs(cmd *cobra.Command, args []string) error {
	serverName := args[0]
	logger.Infof("starting server %s\n", serverName)

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}

	cont, err := cli.ContainerInspect(context.Background(), util.GetContainerName(serverName))
	if err != nil {
		return err
	}

	body, err := cli.ContainerLogs(context.Background(), cont.ID, types.ContainerLogsOptions{
		Follow:     viper.GetBool("follow"),
		ShowStderr: true,
		ShowStdout: true,
		Timestamps: false,
	})
	if err != nil {
		return err
	}
	defer body.Close()

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return nil
}

func SendCommand(serverName string, args []string) error {
	index, serverCfg := config.Cfg.Servers.GetByName(serverName)
	if serverCfg == nil {
		return fmt.Errorf("no server config found for %s", serverName)
	}

	resp, err := http.PostForm(fmt.Sprintf("http://127.0.0.1:4%03d/", index), url.Values{
		"auth-key": {serverCfg.RCON.Password},
		"command":  {strings.Join(args, " ")},
	})
	if err != nil {
		log.Fatalln(err)
	}
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	out, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(out))

	return fmt.Errorf("error during send command to srcds_runner for %s", serverName)
}

func UpdateRCONPassword(serverName string, password string) error {
	index, serverCfg := config.Cfg.Servers.GetByName(serverName)
	if serverCfg == nil {
		return fmt.Errorf("no server config found for %s", serverName)
	}

	resp, err := http.PostForm(fmt.Sprintf("http://127.0.0.1:4%03d/rconPwUpdate", index), url.Values{
		"auth-key": {serverCfg.RCON.Password},
		"password": {password},
	})
	if err != nil {
		log.Fatalln(err)
	}
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	return fmt.Errorf("error during send command to srcds_runner for %s", serverName)
}
