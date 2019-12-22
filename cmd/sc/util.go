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

package main

import (
	"fmt"
	"strings"

	"github.com/galexrt/srcds_controller/pkg/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func checkServers(cmd *cobra.Command, args []string) ([]string, error) {
	var servers []string
	if viper.GetBool(AllServers) || (len(args) > 0 && strings.ToLower(args[0]) == AllServers) {
		for _, srv := range config.Cfg.Servers {
			servers = append(servers, srv.Name)
		}
	} else if len(args) > 0 {
		servers = strings.Split(args[0], ",")
	}

	if len(servers) == 0 {
		return servers, fmt.Errorf("no server(s) given, please provide a server list as the first argument, example: `sc " + cmd.Name() + " SERVER_A,SERVER_B` or `all` instead of the server list")
	}

	for _, server := range servers {
		serverCfg := config.Cfg.Servers.GetByName(server)
		if serverCfg == nil {
			return servers, fmt.Errorf("server %s not found in config", server)
		}
		if !serverCfg.Enabled {
			log.Warningf("server %s is not enabled, skipping ...", server)
		}
	}
	return servers, nil
}
