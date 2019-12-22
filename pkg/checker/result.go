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
type ResultServerList struct {
	sync.RWMutex
	results map[string]map[string]*ResultCounter
}

// Result
type Result struct {
	Check  config.Check
	Server *config.Config
	Return bool
}

// NewResultServerList
func NewResultServerList() *ResultServerList {
	return &ResultServerList{
		results: map[string]map[string]*ResultCounter{},
	}
}

// Add
func (r *ResultServerList) Add(result Result) {
	r.Lock()
	if _, ok := r.results[result.Server.Server.Name]; !ok {
		r.results[result.Server.Server.Name] = map[string]*ResultCounter{}
	}
	if result.Return {
		r.Unlock()
		if _, ok := r.results[result.Server.Server.Name][result.Check.Name]; ok {
			delete(r.results[result.Server.Server.Name], result.Check.Name)
		}
		return
	}
	now := time.Now()
	if _, ok := r.results[result.Server.Server.Name][result.Check.Name]; !ok {
		r.results[result.Server.Server.Name][result.Check.Name] = &ResultCounter{
			Count:     0,
			FirstTime: now,
		}
	}
	r.results[result.Server.Server.Name][result.Check.Name].Count++
	r.results[result.Server.Server.Name][result.Check.Name].LastTime = now

	serverCfg := result.Server
	check := result.Check
	counter := r.results[result.Server.Server.Name][result.Check.Name]
	r.Unlock()

	log.WithField("server", result.Server.Server.Name).Debugf("evaluating result counter for server %s check %s", serverCfg.Server.Name, check.Name)
	log.WithField("server", result.Server.Server.Name).Debugf("current state: count: %d/%d, time: %s - %s", counter.Count, check.Limit.Count, counter.LastTime.Sub(counter.FirstTime), check.Limit.After)
	if (check.Limit.Count != 0 && counter.Count >= check.Limit.Count) ||
		(check.Limit.After != 0 && counter.LastTime.Sub(counter.FirstTime) >= check.Limit.After) {

		log.WithField("server", result.Server.Server.Name).Infof("result counter over limit for server %s check %s", serverCfg.Server.Name, check.Name)

		counter.Count = 0
		now := time.Now()
		counter.FirstTime = now
		counter.LastTime = now

		r.runAction(check, serverCfg)
	} else {
		log.WithField("server", result.Server.Server.Name).Debugf("nothing to do for server %s", serverCfg.Server.Name)
	}
}

func (r *ResultServerList) runAction(check config.Check, serverCfg *config.Config) {
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, action := range check.Limit.Actions {
			switch strings.ToLower(action) {
			case "restart":
				if viper.GetBool("dry-run") {
					log.WithField("server", serverCfg.Server.Name).Debugf("dry-run mode active, server %s restart", serverCfg.Server.Name)
				} else {
					log.WithField("server", serverCfg.Server.Name).Infof("need to restart server %s", serverCfg.Server.Name)
					if err := server.SendCommand(serverCfg, []string{"say", "SRCDS CHECKER RESTART MARKER"}); err != nil {
						log.Error(err)
					}
					if err := server.Restart(serverCfg); err != nil {
						log.WithField("server", serverCfg.Server.Name).Error(err)
					}
					log.WithField("server", serverCfg.Server.Name).Infof("server %s restarted", serverCfg.Server.Name)
				}
			case "log":
				log.WithField("server", serverCfg.Server.Name).Warn("runAction dummy log action")
			}
		}
	}()

	wg.Wait()
	return
}
