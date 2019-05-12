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
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	SRCDSRunnerAuthKeyEnvKey = "SRCDS_RUNNER_AUTH_KEY"
)

var (
	DockerCli *client.Client
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func List() error {
	w := tabwriter.NewWriter(os.Stdout, 1, 0, 1, ' ', tabwriter.Debug)
	fmt.Fprintln(w, "Name\tPort\tContainer Status")
	for _, serverCfg := range config.Cfg.Servers {
		serverName := util.GetContainerName(serverCfg.Name)
		cont, err := DockerCli.ContainerInspect(context.Background(), serverName)
		status := "Not Running"
		if err != nil {
			if !client.IsErrNotFound(err) {
				return err
			}
		}
		if cont.ContainerJSONBase != nil {
			status = cont.State.Status
		}
		fmt.Fprintf(w, "%s\t%d\t%s\n", strings.TrimPrefix(serverName, config.Cfg.Docker.NamePrefix), serverCfg.Port, status)
	}
	return w.Flush()
}

func Start(serverName string) error {
	log.Infof("starting server %s ...\n", serverName)

	if _, serverCfg := config.Cfg.Servers.GetByName(serverName); serverCfg == nil {
		return fmt.Errorf("no server config found for %s", serverName)
	}

	cont, err := DockerCli.ContainerInspect(context.Background(), util.GetContainerName(serverName))
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
		mountDir := serverCfg.MountsDir

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
			fmt.Sprintf("SRCDS_SERVER_NAME=%s", serverCfg.Name),
			fmt.Sprintf("SRCDS_RUNNER_PORT=%d", serverCfg.RunnerPort),
			fmt.Sprintf("%s=%s", SRCDSRunnerAuthKeyEnvKey, util.RandString(128)),
		}
		if serverCfg.OnExitCommand != "" {
			envs = append(envs, fmt.Sprintf("SRCDS_RUNNER_ONEXIT_COMMAND=%s", serverCfg.OnExitCommand))
		}

		contCfg := &container.Config{
			Labels: map[string]string{
				"app":        "gameserver",
				"managed-by": "srcds_controller",
			},
			Env:         envs,
			Cmd:         contArgs,
			AttachStdin: true,
			Tty:         false,
			OpenStdin:   true,
			Hostname:    hostname,
			User:        fmt.Sprintf("%d:%d", serverCfg.RunOptions.UID, serverCfg.RunOptions.GID),
			Image:       config.Cfg.Docker.Image,
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
					Source:   config.FilePath,
					Target:   "/config/config.yaml",
					ReadOnly: true,
				},
				{
					Type:   mount.TypeBind,
					Source: serverDir,
					Target: serverDir,
				},
			},
			NetworkMode: "host",
		}
		if mountDir != "" {
			contHostCfg.Mounts = append(contHostCfg.Mounts, mount.Mount{
				Type:   mount.TypeBind,
				Source: mountDir,
				Target: mountDir,
			})
		}
		if serverCfg.Resources != nil {
			contHostCfg.Resources = *serverCfg.Resources
		}

		netCfg := &network.NetworkingConfig{}
		var resp container.ContainerCreateCreatedBody
		resp, err = DockerCli.ContainerCreate(context.Background(), contCfg, contHostCfg, netCfg, util.GetContainerName(serverName))
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

	log.Infof("started server %s.", serverName)

	return nil
}

func Stop(serverName string) error {
	log.Infof("stopping server %s ...", serverName)

	if _, serverCfg := config.Cfg.Servers.GetByName(serverName); serverCfg == nil {
		return fmt.Errorf("no server config found for %s", serverName)
	}

	cont, err := DockerCli.ContainerInspect(context.Background(), util.GetContainerName(serverName))
	if err != nil {
		if client.IsErrNotFound(err) {
			return nil
		}
		return err
	}

	containerID := cont.ID

	if cont.State.Running {
		log.Infof("sending SIGTERM signal to server %s ...", serverName)
		if err = DockerCli.ContainerKill(context.Background(), containerID, "SIGTERM"); err != nil {
			log.Error(err)
		}
		log.Infof("sent SIGTERM signal to server %s. now waiting at maximum 5 before sending kill signal ...", serverName)
	}

	duration := viper.GetDuration("timeout")
	if err = DockerCli.ContainerStop(context.Background(), containerID, &duration); err != nil {
		return err
	}

	log.Infof("stopped server %s.", serverName)
	return nil
}

func Restart(serverName string) error {
	if err := Stop(serverName); err != nil {
		return err
	}
	if viper.GetBool("remove") {
		if err := Remove(serverName); err != nil {
			return err
		}
	}
	return Start(serverName)
}

func Remove(serverName string) error {
	log.Infof("removing server container %s ...", serverName)

	if _, serverCfg := config.Cfg.Servers.GetByName(serverName); serverCfg == nil {
		return fmt.Errorf("no server config found for %s", serverName)
	}

	cont, err := DockerCli.ContainerInspect(context.Background(), util.GetContainerName(serverName))
	if err != nil {
		return err
	}

	if err = DockerCli.ContainerRemove(context.Background(), cont.ID, types.ContainerRemoveOptions{}); err != nil {
		return err
	}

	log.Infof("removed server container %s.", serverName)
	return nil
}

func Logs(serverName string, since time.Duration, tail int) (io.ReadCloser, io.ReadCloser, error) {
	log.Infof("showing logs of server %s ...\n", serverName)

	if _, serverCfg := config.Cfg.Servers.GetByName(serverName); serverCfg == nil {
		return nil, nil, fmt.Errorf("no server config found for %s", serverName)
	}

	cont, err := DockerCli.ContainerInspect(context.Background(), util.GetContainerName(serverName))
	if err != nil {
		return nil, nil, err
	}

	args := []string{"logs"}
	if viper.GetBool("follow") {
		args = append(args, "-f")
	}

	if since != 0*time.Millisecond {
		args = append(args, fmt.Sprintf("--since=%s", since.String()))
	} else if tail != 0 {
		args = append(args, fmt.Sprintf("--tail=%d", tail))
	}

	args = append(args, cont.ID)

	cmd := exec.Command("docker", args...)

	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	return stdoutIn, stderrIn, nil
}

func SendCommand(serverName string, args []string) error {
	log.Infof("sending command '%s' to server %s ...\n", strings.Join(args, " "), serverName)

	_, serverCfg := config.Cfg.Servers.GetByName(serverName)
	if serverCfg == nil {
		return fmt.Errorf("no server config found for %s", serverName)
	}

	cont, err := GetServerContainer(serverName)
	if err != nil {
		return err
	}

	var authKey string
	for _, env := range cont.Config.Env {
		if strings.HasPrefix(env, fmt.Sprintf("%s=", SRCDSRunnerAuthKeyEnvKey)) {
			authKey = strings.Split(env, "=")[1]
			break
		}
	}

	if authKey == "" {
		authKey = serverCfg.RCON.Password
	}

	resp, err := http.PostForm(fmt.Sprintf("http://127.0.0.1:%d/", serverCfg.RunnerPort), url.Values{
		"auth-key": {authKey},
		"command":  {strings.Join(args, " ")},
	})
	if err != nil {
		return fmt.Errorf("error during command exec send to server %s. %+v", serverName, err)
	}
	if resp.StatusCode == http.StatusOK {
		log.Infof("successfully sent command to server %s.\n", serverName)
		return nil
	}

	out, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(out))

	return fmt.Errorf("error during sending of command to srcds_runner for server %s", serverName)
}

func GetServerContainer(serverName string) (types.ContainerJSON, error) {
	var err error
	var cont types.ContainerJSON

	if _, serverCfg := config.Cfg.Servers.GetByName(serverName); serverCfg == nil {
		return cont, fmt.Errorf("no server config found for %s", serverName)
	}

	cont, err = DockerCli.ContainerInspect(context.Background(), util.GetContainerName(serverName))
	if err != nil {
		return cont, err
	}

	return cont, nil
}

func WaitForConsoleContains(serverName string, pattern string) (bool, error) {
	// TODO Stream the logs and return true if given pattern is found.

	return true, nil
}
