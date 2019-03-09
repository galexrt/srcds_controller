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
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

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

func List(cmd *cobra.Command, args []string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 1, 0, 1, ' ', tabwriter.Debug)
	fmt.Fprintln(w, "Name\tPort\tContainer Status")
	for _, serverCfg := range config.Cfg.Servers {
		serverName := util.GetContainerName(serverCfg.Name)
		cont, err := cli.ContainerInspect(context.Background(), serverName)
		status := "Not Running"
		if err != nil {
			if !client.IsErrNotFound(err) {
				return err
			}
		}
		if cont.ContainerJSONBase != nil {
			status = cont.State.Status
		}
		fmt.Fprintf(w, "%s\t%d\t%s\n", strings.TrimPrefix(serverName, config.Cfg.Docker.NamePrefix+"-"), serverCfg.Port, status)
	}
	return w.Flush()
}

func Start(cmd *cobra.Command, args []string) error {
	serverName := args[0]
	logger.Infof("starting server %s ...\n", serverName)

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
		_, serverCfg := config.Cfg.Servers.GetByName(serverName)
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
			arg = strings.Replace(arg, "%RCON_PASSWORD%", serverCfg.RCON.Password, -1)
			contArgs = append(contArgs, arg)
		}

		envs := []string{
			fmt.Sprintf("SRCDS_RUNNER_PORT=%d", serverCfg.RunnerPort),
			fmt.Sprintf("SRCDS_RUNNER_AUTH_KEY=%s", serverCfg.RCON.Password),
		}
		if serverCfg.OnExitCommand != "" {
			envs = append(envs, fmt.Sprintf("SRCDS_RUNNER_ONEXIT_COMMAND=%s", serverCfg.OnExitCommand))
		}

		contCfg := &container.Config{
			Env:         envs,
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

	logger.Infof("started server %s.", serverName)

	return nil
}

func Stop(cmd *cobra.Command, args []string) error {
	serverName := args[0]
	logger.Infof("stopping server %s ...", serverName)

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
	if err = cli.ContainerStop(context.Background(), containerID, &duration); err != nil {
		return err
	}

	logger.Infof("stopped server %s.", serverName)
	return nil
}

func Restart(cmd *cobra.Command, args []string) error {
	if err := Stop(cmd, args); err != nil {
		return err
	}
	if viper.GetBool("remove") {
		if err := Remove(cmd, args); err != nil {
			return err
		}
	}
	return Start(cmd, args)
}

func Remove(cmd *cobra.Command, args []string) error {
	serverName := args[0]
	logger.Infof("removing server container %s ...", serverName)

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}

	cont, err := cli.ContainerInspect(context.Background(), util.GetContainerName(serverName))
	if err != nil {
		return err
	}

	if err = cli.ContainerRemove(context.Background(), cont.ID, types.ContainerRemoveOptions{}); err != nil {
		return err
	}

	logger.Infof("removed server container %s.", serverName)
	return nil
}

func Logs(cmd *cobra.Command, args []string) (io.ReadCloser, error) {
	serverName := args[0]
	logger.Infof("showing logs of server %s ...\n", serverName)

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	cont, err := cli.ContainerInspect(context.Background(), util.GetContainerName(serverName))
	if err != nil {
		return nil, err
	}

	body, err := cli.ContainerLogs(context.Background(), cont.ID, types.ContainerLogsOptions{
		Follow:     viper.GetBool("follow"),
		ShowStderr: true,
		ShowStdout: true,
		Timestamps: false,
	})
	return body, err
}

func SendCommand(serverName string, args []string) error {
	logger.Infof("sending command '%s' to server %s ...\n", strings.Join(args, " "), serverName)

	_, serverCfg := config.Cfg.Servers.GetByName(serverName)
	if serverCfg == nil {
		return fmt.Errorf("no server config found for %s", serverName)
	}

	resp, err := http.PostForm(fmt.Sprintf("http://127.0.0.1:%d/", serverCfg.RunnerPort), url.Values{
		"auth-key": {serverCfg.RCON.Password},
		"command":  {strings.Join(args, " ")},
	})
	if err != nil {
		return fmt.Errorf("error during command exec send to server %s. %+v", serverName, err)
	}
	if resp.StatusCode == http.StatusOK {
		logger.Infof("successfully sent command to server %s.\n", serverName)
		return nil
	}

	out, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(out))

	return fmt.Errorf("error during sending of command to srcds_runner for server %s", serverName)
}

func UpdateRCONPassword(serverName string, password string) error {
	logger.Infof("updating RCON password for server %s ...\n", serverName)

	_, serverCfg := config.Cfg.Servers.GetByName(serverName)
	if serverCfg == nil {
		return fmt.Errorf("no server config found for %s", serverName)
	}

	resp, err := http.PostForm(fmt.Sprintf("http://127.0.0.1:%d/rconPwUpdate", serverCfg.RunnerPort), url.Values{
		"auth-key": {serverCfg.RCON.Password},
		"password": {password},
	})
	if err != nil {
		return fmt.Errorf("error during RCON password update send to server %s. %+v", serverName, err)
	}
	if resp.StatusCode == http.StatusOK {
		logger.Infof("successfully updated RCON password for server %s.\n", serverName)
		return nil
	}

	return fmt.Errorf("error during send command to srcds_runner for %s", serverName)
}
