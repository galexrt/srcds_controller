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

	"github.com/acarl005/stripansi"
	"github.com/creack/pty"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serverToolsUpdate represents the stop command
var serverToolsUpdate = &cobra.Command{
	Use:               "update",
	Short:             "Update one ore more servers",
	PersistentPreRunE: initDockerCli,
	RunE: func(cmd *cobra.Command, args []string) error {
		servers, err := checkServers(cmd, args)
		if err != nil {
			return err
		}

		errorOccured := false
		for _, serverCfg := range servers {
			logger := log.WithFields(log.Fields{
				"server": serverCfg.Server.Name,
				"path":   serverCfg.Server.Path,
			})

			// Set the server dir as the home, unless otherwise set
			serverHomeDir := serverCfg.Server.Path
			if serverCfg.Server.RunOptions.HomeDir != "" {
				serverHomeDir = serverCfg.Server.RunOptions.HomeDir
			}

			// Base docker Command + Args
			command := "docker"
			commandArgs := []string{
				"run",
				"--interactive",
				"--tty",
				// Set correct user + work dir
				fmt.Sprintf("--user=%d:%d", serverCfg.Server.RunOptions.UID, serverCfg.Server.RunOptions.GID),
				fmt.Sprintf("--workdir=%s", serverCfg.Server.Path),
				fmt.Sprintf("--env=HOME=%s", serverHomeDir),
				// Add volumes
				fmt.Sprintf("--volume=%s:%s", serverCfg.Server.Path, serverCfg.Server.Path),
				fmt.Sprintf("--volume=%s:%s", serverCfg.Server.SteamCMDDir, serverCfg.Server.SteamCMDDir),
				*serverCfg.Docker.Image,
			}

			argAppUpdate := fmt.Sprintf("+app_update %d", serverCfg.Server.GameID)
			beta := viper.GetString("beta")
			if beta != "" {
				argAppUpdate += fmt.Sprintf(" -beta %s", beta)
			}
			argAppUpdate += " validate"

			// steamcmd.sh Command
			steamCmdCommand := []string{
				path.Join(serverCfg.Server.SteamCMDDir, "steamcmd.sh"),
				"+login anonymous",
				fmt.Sprintf("+force_install_dir %s", serverCfg.Server.Path),
				argAppUpdate,
			}
			if viper.GetBool(AllServers) {
				steamCmdCommand = append(steamCmdCommand, args[0:]...)
			} else if len(args) > 1 {
				steamCmdCommand = append(steamCmdCommand, args[1:]...)
			}
			steamCmdCommand = append(steamCmdCommand, "+quit")
			commandArgs = append(commandArgs, strings.Join(steamCmdCommand, " "))

			logger.Debugf("full command: %s", commandArgs)
			logger.Infof("running steamcmd command in container: %s", steamCmdCommand)
			if err := func() error {
				cmd := exec.Command(command, commandArgs...)
				tty, err := pty.Start(cmd)
				if err != nil {
					logger.Errorf("%+v", err)
					return err
				}
				defer func() {
					if tty == nil {
						logger.Debug("failed to close tty as it is already nil")
						return
					}
					if err = tty.Close(); err != nil {
						logger.Debug(err)
						return
					}
				}()

				go func() {
					logger.Debug("beginning to stream logs")
					copyLogs(tty)
				}()

				if err := cmd.Wait(); err != nil {
					logger.Errorf("%+v", err)
					errorOccured = true
				}

				return nil
			}(); err != nil {
				logger.Errorf("%+v", err)
				break
			}
		}

		if errorOccured {
			return fmt.Errorf("error when sending commands")
		}
		return nil
	},
}

func init() {
	serverToolsUpdate.PersistentFlags().String("beta", "", "which branch to install during steamcmd app_install validate")
	viper.BindPFlag("beta", serverToolsUpdate.PersistentFlags().Lookup("beta"))

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
