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
	log "github.com/sirupsen/logrus"
)

func init() {
	checks.Checks["actioreactio"] = Run
}

// Run run a actioreactio check on a config.Server
func Run(check config.Check, srv *config.Config) bool {
	// TODO Run command by Console and check the logs for time X if there is an appropriate response in it

	stdout, stderr, err := server.Logs(srv, 0*time.Second, 0, true)
	if err != nil {
		log.Errorf("error while getting logs from server. %+v", err)
		return false
	}
	defer stdout.Close()
	defer stderr.Close()

	timeout, ok := check.Opts["timeout"]
	if !ok {
		timeout = "5s"
	}
	timeoutDuration, err := time.ParseDuration(timeout)
	if err != nil {
		log.Errorf("failed to parse actioreactio timeout check opts. %+v", err)
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	foundInfo := make(chan bool)
	go func() {
		foundInfo <- checkStreamForString(ctx, stdout, `Unknown command "srcsd_controller_check"`)
	}()
	go func() {
		foundInfo <- checkStreamForString(ctx, stderr, `Unknown command "srcsd_controller_check"`)
	}()

	if err := server.SendCommand(srv, []string{
		"srcds_controller_check",
	}); err != nil {
		log.Errorf("error while sending actioreactio command to server. %+v", err)
		return false
	}

	for {
		select {
		case <-ctx.Done():
			return false
		case found := <-foundInfo:
			if found {
				close(foundInfo)
				return true
			}
		}
	}
}

func checkStreamForString(ctx context.Context, stream io.Reader, search string) bool {
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		text := scanner.Text()
		if strings.Contains(text, search) {
			return true
		}
	}

	if err := scanner.Err(); err != nil {
		log.Errorf("error during logs line scanning. %+v", err)
	}

	return false
}
