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
	"sync"
	"time"

	"github.com/galexrt/srcds_controller/pkg/checks"
	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/server"
	"github.com/imdario/mergo"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

var (
	defaultOpts = config.CheckOpts{
		"timeout": "10s",
	}
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
		logger.Fatalf("failed to merge checks opts and actioreactio check defaults %s", srv.Server.Name)
	}

	timeoutDuration, err := time.ParseDuration(check.Opts["timeout"])
	if err != nil {
		logger.Errorf("failed to parse actioreactio timeout check opts. %+v", err)
		return false
	}

	startTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	cmd, stdout, stderr, err := server.Logs(ctx, srv, 0*time.Second, 5, true)
	if err != nil {
		logger.Errorf("error while getting logs from server. %+v", err)
		return false
	}
	defer stdout.Close()

	wg := &sync.WaitGroup{}

	foundCh := make(chan bool, 1)

	wg.Add(1)
	go func() {
		defer wg.Done()
		cmd.Wait()
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		checkStreamForString(stderr, foundCh, `Unknown command "srcds_controller_check"`)
	}()

	if err := server.SendCommand(srv, []string{
		"srcds_controller_check",
	}); err != nil {
		logger.Errorf("error while sending actioreactio command to server. %+v", err)
		return false
	}

	result := false
	select {
	case <-ctx.Done():
		logger.Errorf("timeout while waiting for actioreactio output (%+v)", time.Now().Sub(startTime))
	case result = <-foundCh:
		logger.Debugf("got a result in time (%+v): %+v", time.Now().Sub(startTime), result)
	}

	wg.Wait()
	close(foundCh)
	return result
}

func checkStreamForString(stream io.ReadCloser, foundCh chan bool, search string) {
	defer stream.Close()
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		text := scanner.Text()
		log.Debugf("checkStreamForString line: %+v", text)
		if strings.Contains(text, search) {
			foundCh <- true
			return
		}
	}

	if err := scanner.Err(); err != nil {
		log.Debugf("error during logs line scanning. %+v", err)
	}
	foundCh <- false
}
