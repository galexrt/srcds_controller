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
	"sync"

	"github.com/acarl005/stripansi"
	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/util"
	"github.com/kr/pty"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serverToolsUpdate represents the stop command
var serverToolsUpdate = &cobra.Command{
	Use:               "update",
	Short:             "Update a gameserver",
	PersistentPreRunE: initDockerCli,
	RunE: func(cmd *cobra.Command, args []string) error {
		var errs util.Errors
		var servers []string
		if viper.GetBool("all") {
			for _, srv := range config.Cfg.Servers {
				servers = append(servers, srv.Name)
			}
		} else {
			servers = viper.GetStringSlice("servers")
		}
		if len(servers) == 0 {
			return fmt.Errorf("no server list given (`--servers=SERVER_A,SERVER_B`) or `--all` flag not given (can also mean that no servers are in the config)")
		}

		// Get current work
		home, err := homedir.Dir()
		if err != nil {
			return err
		}

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
				command := exec.Command(path.Join(home, "steamcmd/steamcd.sh"), commandArgs...)
				tty, err := pty.Start(command)
				if err != nil {
					logger.Fatal(err)
					errs.Lock()
					errs.Errs = append(errs.Errs, err)
					errs.Unlock()
					return
				}
				defer func() {
					if tty == nil {
						logger.Debug("failed to close tty as it is nil")
						return
					}
					if err = tty.Close(); err != nil {
						logger.Debug(err)
						return
					}
				}()

				wg.Add(1)
				go func() {
					defer wg.Done()
					logger.Debug("beginning to stream logs")
					copyLogs(tty)
				}()

				if err := command.Wait(); err != nil {
					errs.Lock()
					errs.Errs = append(errs.Errs, err)
					errs.Unlock()
				}
			}(serverName)
		}
		wg.Wait()

		if len(errs.Errs) > 0 {
			err := errors.New("error occured during server (tools) update")
			for _, erro := range errs.Errs {
				err = errors.Wrap(err, erro.Error())
			}
			return err
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
			logger.Debug("copyLogs: received EOF from given log source")
			return nil
		}
		if err != nil {
			logger.Debug(err)
			return err
		}
	}
}
