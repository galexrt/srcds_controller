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
	"os/exec"
	"time"

	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/util"
	log "github.com/sirupsen/logrus"
)

// Logs show / stream the logs of a server
func Logs(serverCfg *config.Config, since time.Duration, tail int, follow bool) (io.ReadCloser, io.ReadCloser, error) {
	log.Infof("showing logs of server %s ...\n", serverCfg.Server.Name)

	cont, err := DockerCli.ContainerInspect(context.Background(), util.GetContainerName(serverCfg.Docker.NamePrefix, serverCfg.Server.Name))
	if err != nil {
		return nil, nil, err
	}

	args := []string{"logs"}
	if follow {
		args = append(args, "--follow")
	}

	if since != 0*time.Millisecond {
		args = append(args, fmt.Sprintf("--since=%s", since.String()))
	} else if tail != 0 {
		args = append(args, fmt.Sprintf("--tail=%d", tail))
	}

	args = append(args, cont.ID)

	cmd := exec.Command("docker", args...)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	return stdout, stderr, nil
}
