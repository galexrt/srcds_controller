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

	"github.com/acarl005/stripansi"
	"github.com/creack/pty"
	homedir "github.com/mitchellh/go-homedir"
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

		// Get current work
		home, err := homedir.Dir()
		if err != nil {
			return err
		}

		errorOccured := false
		for _, serverCfg := range servers {
			commandArgs := []string{
				"+login anonymous",
				fmt.Sprintf("+force_install_dir %s", serverCfg.Server.Path),
				fmt.Sprintf("+app_update %d", serverCfg.Server.GameID), "validate",
			}

			if viper.GetBool(AllServers) {
				commandArgs = append(commandArgs, args[0:]...)
			} else if len(args) > 1 {
				commandArgs = append(commandArgs, args[1:]...)
			}

			commandArgs = append(commandArgs, "+quit")
			command := exec.Command(path.Join(home, "steamcmd/steamcmd.sh"), commandArgs...)
			tty, err := pty.Start(command)
			if err != nil {
				log.Errorf("%+v", err)
				break
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

			go func() {
				log.Debug("beginning to stream logs")
				copyLogs(tty)
			}()

			if err := command.Wait(); err != nil {
				log.Errorf("%+v", err)
				errorOccured = true
			}
		}

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
