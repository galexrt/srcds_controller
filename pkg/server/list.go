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
	"strings"
	"text/tabwriter"

	"github.com/docker/docker/client"
	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/util"
)

// List list the servers from the config
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
