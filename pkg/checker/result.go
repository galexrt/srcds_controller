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
	"strings"
	"sync"
	"time"

	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// ResultCounter
type ResultCounter struct {
	Count     int64
	FirstTime time.Time
	LastTime  time.Time
}

// ResultServerList
type ResultServerList map[string]map[string]*ResultCounter

// Result
type Result struct {
	Check  config.Check
	Server *config.Server
	Return bool
}

// Add
func (r ResultServerList) Add(result Result) {
	if _, ok := r[result.Server.Name]; !ok {
		r[result.Server.Name] = map[string]*ResultCounter{}
	}
	if !result.Return {
		if _, ok := r[result.Server.Name][result.Check.Name]; !ok {
			r[result.Server.Name][result.Check.Name] = &ResultCounter{
				Count:     0,
				FirstTime: time.Now(),
				LastTime:  time.Now(),
			}
		}
		r[result.Server.Name][result.Check.Name].Count++
		r[result.Server.Name][result.Check.Name].LastTime = time.Now()
	} else {
		if _, ok := r[result.Server.Name][result.Check.Name]; ok {
			delete(r[result.Server.Name], result.Check.Name)
		}
		return
	}

	r.evaluate(r[result.Server.Name][result.Check.Name], result.Check, result.Server)
}

func (r ResultServerList) evaluate(counter *ResultCounter, check config.Check, serverCfg *config.Server) {
	wg := sync.WaitGroup{}
	log.Debugf("evaluating result counter for server %s check %s", serverCfg.Name, check.Name)
	if (check.Limit.Count != 0 && counter.Count >= check.Limit.Count) || (check.Limit.After != 0 && counter.LastTime.Sub(counter.FirstTime) >= check.Limit.After) {
		log.Infof("result counter over limit for server %s check %s", serverCfg.Name, check.Name)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, action := range check.Limit.Actions {
				switch strings.ToLower(action) {
				case "restart":
					if viper.GetBool("dry-run") {
						log.Debugf("dry-run mode active, server %s restart", serverCfg.Name)
					} else {
						log.Infof("need to restart server %s", serverCfg.Name)
						if err := server.SendCommand(serverCfg.Name, []string{"say", "SRCDS CHECKER RESTART MARKER"}); err != nil {
							log.Error(err)
						}
						if err := server.Restart(serverCfg.Name); err != nil {
							log.Error(err)
						}
						log.Infof("server %s restarted", serverCfg.Name)
					}
				}
			}
		}()
		counter.Count = 0
		now := time.Now()
		counter.FirstTime = now
		counter.LastTime = now
	} else {
		log.Debugf("nothing to do for server %s", serverCfg.Name)
	}
	wg.Wait()
	return
}
