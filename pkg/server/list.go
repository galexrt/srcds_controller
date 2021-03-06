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
	"path/filepath"
	"text/tabwriter"

	"github.com/docker/docker/client"
	"github.com/galexrt/srcds_controller/pkg/userconfig"
	"github.com/galexrt/srcds_controller/pkg/util"
)

// List list the servers from the config
func List() error {
	w := tabwriter.NewWriter(os.Stdout, 1, 0, 1, ' ', tabwriter.Debug)
	fmt.Fprintln(w, "Name\tPort\tStatus\tPath")
	for _, serverCfg := range userconfig.Cfg.Servers {
		containerName := util.GetContainerName(serverCfg.Docker.NamePrefix, serverCfg.Server.Name)
		cont, err := DockerCli.ContainerInspect(context.Background(), containerName)
		status := "Not Running"
		if err != nil {
			if !client.IsErrNotFound(err) {
				return err
			}
		}
		if cont.ContainerJSONBase != nil {
			status = cont.State.Status
		}
		fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", serverCfg.Server.Name, serverCfg.Server.Port, status, filepath.Dir(serverCfg.Server.Path))
	}
	return w.Flush()
}
