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

package checker

import (
	"math/rand"
	"sync"
	"time"

	"github.com/galexrt/srcds_controller/pkg/checks"
	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/userconfig"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

var (
	resultCounter = NewResultServerList()
)

type Checker struct {
}

func init() {
	rand.Seed(time.Now().Unix())
}

func New() *Checker {
	return &Checker{}
}

func (c *Checker) Run(stopCh <-chan struct{}) error {
	wg := sync.WaitGroup{}

	resultCh := make(chan Result)

	wg.Add(1)
	go func() {
		defer wg.Done()
		CheckForDockerEvents(stopCh)
	}()

	for _, server := range userconfig.Cfg.Servers {
		wg.Add(2)
		go func(server *config.Config) {
			defer wg.Done()
			for _, check := range server.Server.Checks {
				log.WithFields(logrus.Fields{
					"server": server.Server.Name,
					"check":  check.Name,
				}).Info("starting check")
				go func(check config.Check, server *config.Config) {
					defer wg.Done()
					for {
						log.Debugf("running check %s", check.Name)
						checkResult := checks.Checks[check.Name](check, server)
						resultCh <- Result{
							Check:  check,
							Server: server,
							Return: checkResult,
						}

						splayTime := calculateTimeSplay(server.Checker.Splay.Start, server.Checker.Splay.End)
						waitTime := server.Checker.Interval + splayTime
						log.Debugf("waitTime: %s, splayTime: %s", waitTime, splayTime)

						select {
						case <-time.After(waitTime):
						case <-stopCh:
							return
						}
					}
				}(check, server)
			}
		}(server)
	}

	go func() {
		for {
			select {
			case result := <-resultCh:
				resultCounter.Add(result)
			}
		}
	}()

	for {
		select {
		case <-stopCh:
			wg.Wait()
			close(resultCh)
			log.Info("waitgroup successfully synced")
			return nil
		}
	}
}

func calculateTimeSplay(begin int, end int) time.Duration {
	return time.Duration(rand.Intn(end-begin)+begin) * time.Second
}
