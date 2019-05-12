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

package rcon

import (
	"net"
	"strconv"

	rcon "github.com/galexrt/go-rcon"
	"github.com/galexrt/srcds_controller/pkg/checks"
	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/imdario/mergo"
	log "github.com/sirupsen/logrus"
)

func init() {
	checks.Checks["rcon"] = Run
}

// Run run a rcon check on a config.Server
func Run(check config.Check, server config.Server) bool {
	rconCfg := config.Cfg.Checks["rcon"]
	if err := mergo.Map(&rconCfg, check.Opts); err != nil {
		log.Fatalf("failed to merge checks config and checks opts from server %s", server.Name)
	}

	log.Debugf("connecting to server %s using RCON", server.Name)
	port := strconv.Itoa(server.Port)
	con, err := rcon.Connect(net.JoinHostPort(server.Address, port), &rcon.ConnectOptions{
		RCONPassword: server.RCON.Password,
		Timeout:      rconCfg["timeout"],
	})
	if err != nil {
		log.Errorf("error connecting to server %s using RCON. %+v", server.Name, err)
		return false
	}
	defer con.Close()

	out, err := con.Send("sv_lan")
	if err != nil {
		log.Errorf("error executing rcon `sv_lan` command. %+v", err)
		log.Debugf("rcond `sv_lan` command output: %s", out)
		return false
	}
	log.Debugf("rcond `sv_lan` command output: %s", out)

	return true
}
