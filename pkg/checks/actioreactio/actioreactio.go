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
	log "github.com/sirupsen/logrus"
)

var (
	defaultOpts = map[string]string{
		"timeout": "10s",
	}
)

func init() {
	checks.Checks["actioreactio"] = Run
}

// Run run a actioreactio check on a config.Server
func Run(check config.Check, srv *config.Config) bool {
	if err := mergo.Map(&check.Opts, defaultOpts); err != nil {
		log.Fatalf("failed to merge checks opts and rcon check defaults %s", srv.Server.Name)
	}

	stdout, _, err := server.Logs(srv, 0*time.Second, 1, true)
	if err != nil {
		log.Errorf("error while getting logs from server. %+v", err)
		return false
	}
	defer stdout.Close()

	timeoutDuration, err := time.ParseDuration(check.Opts["timeout"])
	if err != nil {
		log.Errorf("failed to parse actioreactio timeout check opts. %+v", err)
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	wg := sync.WaitGroup{}
	foundInfo := make(chan bool)
	defer close(foundInfo)

	wg.Add(1)
	go func() {
		defer wg.Done()
		foundInfo <- checkStreamForString(ctx, stdout, `Unknown command "srcds_controller_check"`)
	}()

	if err := server.SendCommand(srv, []string{
		"\nsrcds_controller_check",
	}); err != nil {
		log.Errorf("error while sending actioreactio command to server. %+v", err)
		return false
	}

	for {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				log.Errorf("ctx error in actioreactio check timeout. %+v", err)
			}
			return false
		case found := <-foundInfo:
			wg.Wait()
			return found
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
