/*
Copyright 2018 Alexander Trost <galexrt@googlemail.com>

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

	"github.com/coreos/pkg/capnslog"
	rcon "github.com/galexrt/go-rcon"
	"github.com/galexrt/srcds_controller/pkg/checks"
	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/imdario/mergo"
)

var logger = capnslog.NewPackageLogger("github.com/galexrt/srcds_controller", "pkg/checks/rcon")

func init() {
	checks.Checks["rcon"] = Run
}

// Run run a rcon check on a config.Server
func Run(check config.Check, server config.Server) bool {
	rconCfg := config.Cfg.Checks["rcon"]
	if err := mergo.Map(&rconCfg, check.Opts); err != nil {
		logger.Fatalf("failed to merge checks config and checks opts from server %s", server.Name)
	}

	logger.Debugf("connecting to server %s using RCON", server.Name)
	port := strconv.Itoa(server.Port)
	con, err := rcon.Connect(net.JoinHostPort(server.Address, port), &rcon.ConnectOptions{
		RCONPassword: rconCfg["password"],
		Timeout:      rconCfg["timeout"],
	})
	if err != nil {
		logger.Errorf("error connecting to server %s using RCON. %+v", server.Name, err)
		return false
	}
	defer con.Close()

	out, err := con.Send("hostname")
	if err != nil {
		logger.Errorf("error executing rcon `hostname` command. %+v", err)
		logger.Debugf("rcond `hostname` command output: %s", out)
		return false
	}
	logger.Debugf("rcond `hostname` command output: %s", out)

	return true
}
