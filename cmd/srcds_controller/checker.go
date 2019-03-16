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

	"github.com/galexrt/srcds_controller/pkg/checker"
	"github.com/spf13/cobra"

	// Import RCON check
	_ "github.com/galexrt/srcds_controller/pkg/checks/rcon"
)

// checkerCmd represents the checker command
var checkerCmd = &cobra.Command{
	Use:    "checker",
	Short:  "Run the srcd server checker",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		stopCh := make(chan struct{})
		sigCh := make(chan os.Signal)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		logger.Infof("running checker")
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
	rootCmd.AddCommand(checkerCmd)
}
