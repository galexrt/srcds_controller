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

package actionexec

import (
	"os/exec"

	"github.com/coreos/pkg/capnslog"
	"github.com/galexrt/srcds_controller/pkg/config"
)

var logger = capnslog.NewPackageLogger("github.com/galexrt/srcds_controller", "pkg/actionexec")

// RunAction run a command in shell
func RunAction(action string, server config.Server) {
	logger.Infof("running '%s'", action)
	out, err := exec.Command("bash", "-c", action).Output()
	if err != nil {
		logger.Errorf("error running '%s'", action)
	}
	logger.Debugf("output: %s", out)
}
