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
	"github.com/docker/docker/api/types/container"
)

// Servers list of Server
type Servers []*Server

// Server config/info for a server
type Server struct {
	Name          string               `yaml:"name"`
	Enabled       bool                 `yaml:"enabled"`
	Address       string               `yaml:"address"`
	Port          int                  `yaml:"port"`
	MountsDir     string               `yaml:"mountsDir"`
	Command       string               `yaml:"command"`
	Flags         []string             `yaml:"flags"`
	MapSelection  *MapSelection        `yaml:"mapSelection"`
	RCON          *RCON                `yaml:"rcon"`
	Checks        []Check              `yaml:"checks"`
	OnExitCommand string               `yaml:"onExitCommand"`
	GameID        int64                `yaml:"gameID"`
	Resources     *container.Resources `yaml:"resources,omitempty"`
	RunOptions    RunOptions           `yaml:"runOptions"`
	ACL           *ACL                 `yaml:"acl"`
	SteamCMDDir   string               `yaml:"steamCMDDir"`
	Path          string
}

// RCON rcon info
type RCON struct {
	Password string `yaml:"password"`
}

// ACL ACL info
type ACL struct {
	Users  []int `yaml:"users"`
	Groups []int `yaml:"groups"`
}

// RunOptions run options such as user and group id to run the server as.
type RunOptions struct {
	UID     int    `yaml:"uid"`
	GID     int    `yaml:"gid"`
	HomeDir string `yaml:"homeDir"`
}

// MapSelection map selection config
type MapSelection struct {
	Enabled     bool   `yaml:"enabled"`
	FileFilter  string `yaml:"fileFilter"`
	FallbackMap string `yaml:"fallbackMap"`
}
