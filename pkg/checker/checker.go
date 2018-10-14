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

package checker

import (
	"math/rand"
	"sync"
	"time"

	"github.com/coreos/pkg/capnslog"
	"github.com/galexrt/srcds_controller/pkg/checks"
	"github.com/galexrt/srcds_controller/pkg/config"
)

var (
	logger        = capnslog.NewPackageLogger("github.com/galexrt/srcds_controller", "pkg/checker")
	resultCounter = ResultServerList{}
)

type Checker struct {
}

func New() *Checker {
	return &Checker{}
}

func (c *Checker) Run(stopCh <-chan struct{}) error {
	wg := sync.WaitGroup{}

	resultCh := make(chan Result)

	for _, server := range config.Cfg.Servers {
		wg.Add(1)
		go func(server config.Server, stopCh <-chan struct{}) {
			defer wg.Done()
			for _, check := range server.Checks {
				logger.Infof("starting check %s for server %s", server.Name, check.Name)
				wg.Add(1)
				go func(check config.Check, server config.Server) {
					defer wg.Done()
					for {
						splayTime := calculateTimeSplay(config.Cfg.Checker.Splay.Start, config.Cfg.Checker.Splay.End)
						waitTime := config.Cfg.Checker.Interval + splayTime
						logger.Debugf("waitTime: %s, splayTime: %s", waitTime, splayTime)

						select {
						case <-time.After(waitTime):
							logger.Debugf("running check %s", check.Name)
							resultCh <- Result{
								Check:  check,
								Server: server,
								Return: checks.Checks[check.Name](check, server),
							}
						case <-stopCh:
							return
						}
					}
				}(check, server)
			}
		}(server, stopCh)
	}

	go func() {
		for {
			select {
			case result := <-resultCh:
				resultCounter.Add(result)
			case <-stopCh:
				return
			}
		}
	}()

	logger.Infof("waiting for signal")

	select {
	case <-stopCh:
	}
	logger.Info("signal received, waiting on waitgroup ...")
	wg.Wait()
	logger.Info("waitgroup successfully synced")
	return nil
}

func calculateTimeSplay(begin int, end int) time.Duration {
	rand.Seed(time.Now().Unix())
	return time.Duration(rand.Intn(end-begin)+begin) * time.Second
}
