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

package dummy

import (
	"github.com/galexrt/srcds_controller/pkg/checks"
	"github.com/galexrt/srcds_controller/pkg/config"
)

func init() {
	checks.Checks["dummy"] = Run
}

// Run run a dummy check on a config.Server
func Run(check config.Check, server *config.Config) bool {
	return false
}
