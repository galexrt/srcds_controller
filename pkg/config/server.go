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

package config

import (
	"strings"

	"github.com/docker/docker/api/types/container"
)

// Servers list of Server
type Servers []*Server

// Server config/info for a server
type Server struct {
	Name          string               `yaml:"name"`
	Address       string               `yaml:"address"`
	Port          int                  `yaml:"port"`
	RunnerPort    int                  `yaml:"runnerPort"`
	Path          string               `yaml:"path"`
	MountsDir     string               `yaml:"mountsDir"`
	Flags         []string             `yaml:"flags"`
	RunOptions    RunOptions           `yaml:"runOptions"`
	RCON          RCON                 `yaml:"rcon"`
	Checks        []Check              `yaml:"checks"`
	OnExitCommand string               `yaml:"onExitCommand"`
	Enabled       bool                 `yaml:"enabled"`
	GameID        int64                `yaml:"gameID"`
	Resources     *container.Resources `yaml:"resources,omitempty"`
}

// RunOptions run options such as user and group id to run the server as.
type RunOptions struct {
	UID int `yaml:"uid"`
	GID int `yaml:"gid"`
}

// RCON rcon info
type RCON struct {
	Password string `yaml:"password"`
}

// GetByName return server from list by name
func (s Servers) GetByName(name string) (int, *Server) {
	name = strings.ToLower(name)
	for index, server := range s {
		if strings.ToLower(server.Name) == name {
			return index, server
		}
	}
	return -1, nil
}
