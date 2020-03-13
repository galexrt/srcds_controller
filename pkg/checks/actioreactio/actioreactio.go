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

package actioreactio

import (
	"bufio"
	"context"
	"io"
	"strings"
	"time"

	"github.com/galexrt/srcds_controller/pkg/checks"
	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/server"
	"github.com/imdario/mergo"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

var (
	defaultOpts = map[string]string{
		"timeout": "10s",
	}
	foundCh = make(chan bool, 1)
)

func init() {
	checks.Checks["actioreactio"] = Run
}

// Run run a actioreactio check on a config.Server
func Run(check config.Check, srv *config.Config) bool {
	logger := log.WithFields(logrus.Fields{
		"server": srv.Server.Name,
	})

	if err := mergo.Map(&check.Opts, defaultOpts); err != nil {
		logger.Fatalf("failed to merge checks opts and rcon check defaults %s", srv.Server.Name)
	}

	timeoutDuration, err := time.ParseDuration(check.Opts["timeout"])
	if err != nil {
		logger.Errorf("failed to parse actioreactio timeout check opts. %+v", err)
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	stdout, stderr, err := server.Logs(ctx, srv, 0*time.Second, 5, true)
	if err != nil {
		logger.Errorf("error while getting logs from server. %+v", err)
		return false
	}
	stdout.Close()
	defer stderr.Close()

	go checkStreamForString(stderr, `Unknown command "srcds_controller_check"`)

	if err := server.SendCommand(srv, []string{
		"srcds_controller_check",
	}); err != nil {
		logger.Errorf("error while sending actioreactio command to server. %+v", err)
		return false
	}

	select {
	case <-ctx.Done():
		logger.Errorf("timeout while waiting for actioreactio output")
		return false
	case result := <-foundCh:
		logger.Infof("got a result in time: %+v", result)
		return result
	}
}

func checkStreamForString(stream io.Reader, search string) {
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.Contains(text, search) {
			foundCh <- true
			return
		}
	}

	if err := scanner.Err(); err != nil {
		log.Errorf("error during logs line scanning. %+v", err)
	}
	foundCh <- false
}
