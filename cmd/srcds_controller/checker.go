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
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/docker/docker/client"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	// Import RCON check

	"github.com/galexrt/srcds_controller/pkg/checker"
	_ "github.com/galexrt/srcds_controller/pkg/checks/rcon"
	"github.com/galexrt/srcds_controller/pkg/server"
)

// checkerCmd represents the checker command
var checkerCmd = &cobra.Command{
	Use:               "checker",
	Short:             "Run the srcds server checker",
	Hidden:            true,
	PersistentPreRunE: initDockerCli,
	Run: func(cmd *cobra.Command, args []string) {
		stopCh := make(chan struct{})
		sigCh := make(chan os.Signal)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		log.Infof("running checker")

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			checker.New().Run(stopCh)
		}()

		<-sigCh
		close(stopCh)
		wg.Wait()
	},
}

func init() {
	checkerCmd.PersistentFlags().Bool("dry-run", false, "dry run mode")
	viper.BindPFlag("dry-run", checkerCmd.PersistentFlags().Lookup("dry-run"))
	rootCmd.AddCommand(checkerCmd)
}

func initDockerCli(cmd *cobra.Command, args []string) error {
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}
	server.DockerCli = cli
	return err
}
