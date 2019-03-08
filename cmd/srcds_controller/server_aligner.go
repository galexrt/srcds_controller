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
	"log"

	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/server"
	"github.com/spf13/cobra"
)

// serverAlignerCmd represents the aligner command
var serverAlignerCmd = &cobra.Command{
	Use:   "aligner",
	Short: "'Align' server in regards to rcon password, logecho and others.",
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, serverCfg := range config.Cfg.Servers {
			//server.SendCommand(serverCfg.Name, "sv_logecho 1")
			log.Printf("aligning server %s ...", serverCfg.Name)
			if err := server.UpdateRCONPassword(serverCfg.Name, serverCfg.RCON.Password); err != nil {
				return err
			}
			log.Printf("aligned server %s.", serverCfg.Name)
		}

		return nil
	},
}

func init() {
	serverCmd.AddCommand(serverAlignerCmd)
}
