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
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/andygrunwald/cachet"
	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/server"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	cachetDownIncidentName    = "%s wurde neugestartet!"
	cachetDownIncidentMessage = `%s wurde neugestartet aufgrund von automatischen Server Überwachungsmechanismen.

Sollte der Server aufgrund eines aufgetretenden Server Crash automatisch neugestartet worden sein, sollte dieser in den nächsten 1-2 Minuten wieder erreichbar sein.`
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
			case "log":
				log.WithField("server", serverCfg.Server.Name).Warn("runAction dummy log action")
			case "restart":
				r.restartAction(check, serverCfg)
			case "cachet":
				r.cachetAction(check, serverCfg)
			}
		}
	}()

	wg.Wait()
	return
}

func (r *ResultServerList) restartAction(check config.Check, serverCfg *config.Config) {
	if viper.GetBool("dry-run") {
		log.WithField("server", serverCfg.Server.Name).Infof("dry-run mode active, server %s restart", serverCfg.Server.Name)
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
}

func (r *ResultServerList) cachetAction(check config.Check, serverCfg *config.Config) {
	logger := log.WithField("server", serverCfg.Server.Name)
	if viper.GetBool("dry-run") {
		logger.Infof("dry-run mode active, server %s cachet incident or incident update", serverCfg.Server.Name)
		return
	}
	cachetURL := viper.GetString("cachet-url")
	if cachetURL == "" {
		logger.Error("no cachet URL given to controller")
		return
	}
	cachetToken := viper.GetString("cachet-token")
	if cachetToken == "" {
		logger.Error("no cachet API token given to controller")
		return
	}

	if check.Limit == nil {
		logger.Error("no cachet API token given to controller")
		return
	}
	componentIDOpt, ok := check.Limit.ActionOpts["cachetComponentID"]
	if !ok || componentIDOpt == "" {
		logger.Error("no cachet component ID found for server")
		return
	}
	componentID, err := strconv.Atoi(componentIDOpt)
	if err != nil {
		logger.Errorf("failed to convert cachet component ID string to integer. %+v", err)
		return
	}

	client, err := cachet.NewClient(cachetURL, nil)
	if err != nil {
		logger.Error("failed to create cachet API client")
		return
	}
	client.Authentication.SetTokenAuth(cachetToken)

	pong, resp, err := client.General.Ping()
	if err != nil {
		logger.Error("failed to ping cachet API")
		return
	}

	logger.WithFields(logrus.Fields{"status": resp.Status, "response": pong}).Debug("cachet pinged sucessful")

	component, _, err := client.Components.Get(componentID)
	if err != nil {
		logger.Errorf("failed to get component (ID: %d) from cachet API. %+v", componentID, err)
		return
	}

	incidentResp, _, err := client.Incidents.GetAll(&cachet.IncidentsQueryParams{
		ComponentID: componentID,
		Visible:     cachet.ComponentGroupVisibilityPublic,
	})
	if err != nil {
		logger.Errorf("failed to get incidents from cachet API. %+v", err)
		return
	}

	if len(incidentResp.Incidents) > 0 {
		var incident cachet.Incident
		for _, incident = range incidentResp.Incidents {
			if strings.HasPrefix(incident.Name, fmt.Sprintf(cachetDownIncidentName, component.Name)) {
				break
			}
		}

		updated, err := time.Parse("2020-01-08 21:38:45", incident.UpdatedAt)
		if err != nil {
			logger.Errorf("failed to parse incident updated at time. %+v", err)
			return
		}
		// Don't spam the Status page with restarted messages
		if time.Now().Add(1*time.Hour).Sub(updated) > 60*time.Minute {
			return
		}
	}

	newIncident := &cachet.Incident{
		Name:            fmt.Sprintf(cachetDownIncidentName, component.Name),
		Visible:         cachet.IncidentVisibilityPublic,
		ComponentID:     componentID,
		ComponentStatus: cachet.ComponentStatusMajorOutage,
		Status:          cachet.IncidentStatusInvestigating,
		Message:         fmt.Sprintf(cachetDownIncidentMessage, component.Name),
		Notify:          false,
	}
	if _, _, err := client.Incidents.Create(newIncident); err != nil {
		logger.Errorf("failed to create new incident. %+v", err)
		return
	}
}
