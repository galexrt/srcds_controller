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
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"

	"github.com/acarl005/stripansi"
	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/kr/pty"
	homedir "github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serverToolsUpdate represents the stop command
var serverToolsUpdate = &cobra.Command{
	Use:               "update",
	Short:             "Update a gameserver",
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

		// Get current work
		home, err := homedir.Dir()
		if err != nil {
			return err
		}

		errorOccured := false
		wg := sync.WaitGroup{}
		for _, serverName := range servers {
			_, serverCfg := config.Cfg.Servers.GetByName(serverName)
			if serverCfg == nil {
				return fmt.Errorf("no server config found for %s", serverName)
			}

			commandArgs := []string{
				"+login anonymous",
				fmt.Sprintf("+force_install_dir %s", serverCfg.Path),
				"+app_update 4020", "validate",
				"+quit",
			}
			wg.Add(1)
			go func(serverName string) {
				defer wg.Done()
				command := exec.Command(path.Join(home, "steamcmd/steamcmd.sh"), commandArgs...)
				tty, err := pty.Start(command)
				if err != nil {
					log.Errorf("%+v", err)
					errorOccured = true
					return
				}
				defer func() {
					if tty == nil {
						log.Debug("failed to close tty as it is nil")
						return
					}
					if err = tty.Close(); err != nil {
						log.Debug(err)
						return
					}
				}()

				wg.Add(1)
				go func() {
					defer wg.Done()
					log.Debug("beginning to stream logs")
					copyLogs(tty)
				}()

				if err := command.Wait(); err != nil {
					log.Errorf("%+v", err)
					errorOccured = true
				}
			}(serverName)
		}
		wg.Wait()

		if errorOccured {
			return fmt.Errorf("error when sending commands")
		}
		return nil
	},
}

func init() {
	serverToolsCmd.AddCommand(serverToolsUpdate)
}

func copyLogs(r io.Reader) error {
	buf := make([]byte, 512)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			os.Stdout.Write([]byte(
				stripansi.Strip(
					string(buf[0:n]),
				),
			),
			)
		}
		if err == io.EOF {
			log.Debug("copyLogs: received EOF from given log source")
			return nil
		}
		if err != nil {
			log.Debug(err)
			return err
		}
	}
}
