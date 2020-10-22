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

	"github.com/fatih/color"
	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/userconfig"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func checkServers(cmd *cobra.Command, args []string) ([]*config.Config, error) {
	var servers []*config.Config
	if viper.GetBool(AllServers) || (len(args) > 0 && strings.ToLower(args[0]) == AllServers) {
		for _, server := range userconfig.Cfg.Servers {
			servers = append(servers, server)
		}
	} else if len(args) > 0 {
		for _, server := range strings.Split(args[0], ",") {
			cfg, ok := userconfig.Cfg.Servers[server]
			if !ok {
				return servers, fmt.Errorf("servers %s not found", server)
			}
			servers = append(servers, cfg)
		}
	}

	if len(servers) == 0 {
		return servers, fmt.Errorf("no server(s) given, please provide a server list as the first argument, example: `" + cmd.CommandPath() + " SERVER_A,SERVER_B` or `all` instead of the server list")
	}

	return servers, nil
}

func colorMessage(msg string) string {
	// Red
	if strings.Contains(msg, "[ERROR") || strings.HasPrefix(msg, "!! ") {
		msg = color.RedString(msg)
	}
	// Magenta
	if strings.Contains(msg, "[UH-OH!]") {
		msg = color.MagentaString(msg)
	}
	// Blue
	if strings.HasPrefix(msg, "lua_run ") {
		msg = color.BlueString(msg)
	}
	// Cyan
	if strings.HasPrefix(msg, "ServerLog: ") || strings.HasPrefix(msg, "L ") {
		msg = color.CyanString(msg)
	}
	// Green
	if strings.Contains(msg, " connected (") {
		msg = color.GreenString(msg)
	}
	// Yellow
	if strings.Contains(msg, " was kicked because they are on the ban list") {
		msg = color.YellowString(msg)
	}

	return msg
}
