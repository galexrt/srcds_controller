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
	"bufio"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/galexrt/srcds_controller/pkg/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serverLogsCmd represents the logs command
var serverLogsCmd = &cobra.Command{
	Use:               "logs",
	Short:             "Show logs of one or more servers",
	PersistentPreRunE: initDockerCli,
	RunE: func(cmd *cobra.Command, args []string) error {
		servers, err := checkServers(cmd, args)
		if err != nil {
			return err
		}

		var wg sync.WaitGroup
		errors := make(chan error)
		outChan := make(chan string)
		doneCh := make(chan struct{})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		for _, serverCfg := range servers {
			cmd, stdin, stderr, err := server.Logs(ctx, serverCfg, viper.GetDuration("since"), viper.GetInt("tail"), viper.GetBool("follow"))
			if err != nil {
				return err
			}
			if stdin == nil || stderr == nil {
				return fmt.Errorf("server.Logs returned no response. something is wrong")
			}

			wg.Add(3)
			go func() {
				defer wg.Done()
				cmd.Wait()
			}()
			go func(serverName string, stream io.ReadCloser) {
				defer wg.Done()
				defer stdin.Close()

				scanner := bufio.NewScanner(stream)
				for scanner.Scan() {
					msg := scanner.Text()
					msg = colorMessage(msg)
					if len(servers) > 1 {
						msg = fmt.Sprintf("%s: %s", serverName, msg)
					}
					outChan <- msg
				}
				if scanner.Err() != nil {
					errors <- scanner.Err()
					return
				}
			}(serverCfg.Server.Name, stdin)
			go func(serverName string, stream io.ReadCloser) {
				defer wg.Done()
				defer stderr.Close()

				scanner := bufio.NewScanner(stream)
				for scanner.Scan() {
					msg := scanner.Text()
					msg = colorMessage(msg)
					if len(servers) > 1 {
						msg = fmt.Sprintf("%s: %s", serverName, msg)
					}
					if viper.GetBool("debug") {
						outChan <- msg
					}
				}
				if scanner.Err() != nil {
					errors <- scanner.Err()
					return
				}
			}(serverCfg.Server.Name, stderr)
		}

		go func() {
			wg.Wait()
			close(doneCh)
		}()

		for {
			select {
			case out := <-outChan:
				fmt.Println(out)
			case erro := <-errors:
				return erro
			case <-doneCh:
				return nil
			}
		}
	},
}

func init() {
	serverLogsCmd.PersistentFlags().BoolP("follow", "f", true, "Follow the log stream")
	serverLogsCmd.PersistentFlags().DurationP("since", "s", 0*time.Millisecond, "Since when logs should be shown (e.g., 10m will show logs from last 10 minutes to now)")
	serverLogsCmd.PersistentFlags().IntP("tail", "t", 125, "How many lines to show from the past")
	viper.BindPFlag("follow", serverLogsCmd.PersistentFlags().Lookup("follow"))
	viper.BindPFlag("since", serverLogsCmd.PersistentFlags().Lookup("since"))
	viper.BindPFlag("tail", serverLogsCmd.PersistentFlags().Lookup("tail"))

	rootCmd.AddCommand(serverLogsCmd)
}
