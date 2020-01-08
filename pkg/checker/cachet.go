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
	"strings"
	"time"

	"github.com/andygrunwald/cachet"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

const (
	cachetUpIncidentName    = "%s sollte wieder verfügbar sein"
	cachetUpIncidentMessage = `%s.`
)

func cachetStartupIncident(cachetURL string, cachetToken string, componentID int) {
	client, err := cachet.NewClient(cachetURL, nil)
	if err != nil {
		log.Error("failed to create cachet API client")
		return
	}
	client.Authentication.SetTokenAuth(cachetToken)

	pong, resp, err := client.General.Ping()
	if err != nil {
		log.Error("failed to ping cachet API")
		return
	}

	log.WithFields(logrus.Fields{"status": resp.Status, "response": pong}).Debug("cachet pinged sucessful")

	component, _, err := client.Components.Get(componentID)
	if err != nil {
		log.Error("failed to get component (ID: %d) from cachet API. %+v", componentID, err)
		return
	}

	incidentResp, _, err := client.Incidents.GetAll(&cachet.IncidentsQueryParams{
		ComponentID: componentID,
		Visible:     cachet.ComponentGroupVisibilityPublic,
	})
	if err != nil {
		log.Error("failed to get incidents from cachet API. %+v", err)
		return
	}

	if len(incidentResp.Incidents) > 0 {
		var incident cachet.Incident
		for _, incident = range incidentResp.Incidents {
			if strings.HasPrefix(incident.Name, fmt.Sprintf(cachetUpIncidentName, component.Name)) {
				break
			}
		}

		updated, err := time.Parse("2020-01-08 21:38:45", incident.UpdatedAt)
		if err != nil {
			log.Errorf("failed to parse incident updated at time. %+v", err)
			return
		}
		// Don't spam the Status page with restarted messages
		if time.Now().Add(1*time.Hour).Sub(updated) > 60*time.Minute {
			return
		}
	}

	newIncident := &cachet.Incident{
		Name:            fmt.Sprintf(cachetUpIncidentName, component.Name),
		Visible:         cachet.IncidentVisibilityPublic,
		ComponentID:     componentID,
		ComponentStatus: cachet.ComponentStatusOperational,
		Status:          cachet.IncidentStatusFixed,
		Message:         fmt.Sprintf(cachetUpIncidentMessage, component.Name),
		Notify:          false,
	}
	if _, _, err := client.Incidents.Create(newIncident); err != nil {
		log.Errorf("failed to create new incident. %+v", err)
		return
	}
}
