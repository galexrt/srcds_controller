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

package server

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// SendCommand sends a command to a server
func SendCommand(serverCfg *config.Config, args []string) error {
	log.Infof("sending command '%s' to server %s", strings.Join(args, " "), serverCfg.Server.Name)

	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", path.Join(serverCfg.Server.Path, ".srcds_runner.sock"))
			},
		},
	}

	resp, err := httpc.PostForm("http://unixlocalhost/", url.Values{
		"command": {strings.Join(args, " ")},
	})
	if err != nil {
		return fmt.Errorf("error during command exec send to server %s. %+v", serverCfg.Server.Name, err)
	}
	if resp.StatusCode == http.StatusOK {
		log.Infof("successfully sent command to server %s", serverCfg.Server.Name)
		return nil
	}

	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "failed to read body from send command response")
	}
	defer resp.Body.Close()

	return fmt.Errorf("error during sending of command to srcds_runner for server %s (response body: %s)", serverCfg.Server.Name, strings.ReplaceAll(string(out), "\n", "\\n"))
}
