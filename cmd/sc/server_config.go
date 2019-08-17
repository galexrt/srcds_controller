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
	"io/ioutil"
	"strings"

	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

// serverSrvCcfgCmd represents the stop command
var serverSrvCcfgCmd = &cobra.Command{
	Use:               "srvcfg",
	Hidden:            true,
	Short:             "Change certain server config options",
	PersistentPreRunE: initDockerCli,
	RunE: func(cmd *cobra.Command, args []string) error {
		var servers []string
		if viper.GetBool(AllServers) || (len(args) > 0 && strings.ToLower(args[0]) == AllServers) {
			for _, srv := range config.Cfg.Servers {
				servers = append(servers, srv.Name)
			}
		} else if len(args) > 0 {
			servers = strings.Split(args[0], ",")
		}
		if len(servers) == 0 {
			return fmt.Errorf("no server(s) given, please put a server list as the first argument, example: `sc " + cmd.Name() + " SERVER_A,SERVER_B` or `all` instead of the server list")
		}

		for _, server := range servers {
			if _, serverCfg := config.Cfg.Servers.GetByName(server); serverCfg == nil {
				return fmt.Errorf("server %s not found in config", server)
			}
		}

		key := viper.GetString("key")
		value := viper.GetString("value")

		if len(key) == 0 || len(value) == 0 {
			return fmt.Errorf("can't have key and / or value empty")
		}

		switch key {
		case "rconpassword":
		default:
			return fmt.Errorf("unsupported config key %s", key)
		}

		for _, serverName := range servers {
			_, serverCfg := config.Cfg.Servers.GetByName(serverName)
			switch key {
			case "rconpassword":
				serverCfg.RCON.Password = value
			}
		}

		out, err := yaml.Marshal(config.Cfg)
		if err != nil {
			return err
		}

		if err := ioutil.WriteFile(cfgFile, out, 0644); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	serverSrvCcfgCmd.PersistentFlags().String("key", "", "Key to update")
	serverSrvCcfgCmd.PersistentFlags().String("value", "true", "Value to update")
	viper.BindPFlag("key", serverSrvCcfgCmd.PersistentFlags().Lookup("key"))
	viper.BindPFlag("value", serverSrvCcfgCmd.PersistentFlags().Lookup("value"))
	rootCmd.AddCommand(serverSrvCcfgCmd)
}
