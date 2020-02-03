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

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	// Import checks
	"github.com/galexrt/go-rcon"
	_ "github.com/galexrt/srcds_controller/pkg/checks/actioreactio"
	_ "github.com/galexrt/srcds_controller/pkg/checks/rcon"

	"github.com/galexrt/srcds_controller/pkg/checker"
)

// checkerCmd represents the checker command
var checkerCmd = &cobra.Command{
	Use:               "checker",
	Short:             "Run the srcds server checker",
	Hidden:            true,
	PersistentPreRunE: initDockerCli,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Infof("log level set to %s", viper.GetString("log-level"))
		level, err := log.ParseLevel(viper.GetString("log-level"))
		if err != nil {
			return err
		}
		log.SetLevel(level)
		if viper.GetBool("debug") {
			rcon.SetLog(log.WithField("pkg", "go-rcon").Logger)
		}

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

		return nil
	},
}

func init() {
	checkerCmd.PersistentFlags().Bool("dry-run", true, "dry run mode")
	checkerCmd.PersistentFlags().String("log-level", "INFO", "log level")
	checkerCmd.PersistentFlags().Bool("debug", false, "debug mode")
	checkerCmd.PersistentFlags().String("cachet-url", "", "Cachet Status page API v1 URL")
	checkerCmd.PersistentFlags().String("cachet-token", "", "Cachet Status page API token")

	viper.BindPFlag("dry-run", checkerCmd.PersistentFlags().Lookup("dry-run"))
	viper.BindPFlag("log-level", checkerCmd.PersistentFlags().Lookup("log-level"))
	viper.BindPFlag("debug", checkerCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("cachet-url", checkerCmd.PersistentFlags().Lookup("cachet-url"))
	viper.BindPFlag("cachet-token", checkerCmd.PersistentFlags().Lookup("cachet-token"))

	rootCmd.AddCommand(checkerCmd)
}
