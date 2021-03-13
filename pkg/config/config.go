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
	"fmt"
	"path"
	"time"

	"github.com/galexrt/srcds_controller/pkg/util"
	"github.com/mitchellh/go-homedir"
)

// Cfg variables holding the Config
var (
	Cfg *Config
)

// Config config file struct
type Config struct {
	General *General             `yaml:"general"`
	Docker  *Docker              `yaml:"docker"`
	Server  *Server              `yaml:"server"`
	Checker *Checker             `yaml:"checker"`
	Checks  map[string]CheckOpts `yaml:"checks"`
}

// Verify verify the config file
func (c *Config) Verify() error {
	// Checker
	if c.Checker == nil {
		c.Checker = &Checker{}
	}
	if c.Checker.Interval == 0 {
		c.Checker.Interval = 30 * time.Second
	}
	if c.Checker.Splay == nil {
		c.Checker.Splay = &Splay{
			Start: 0,
			End:   20,
		}
	}

	// Docker
	if c.Docker == nil {
		c.Docker = &Docker{}
	}
	if c.Docker.Image == nil {
		c.Docker.Image = util.StringPointer("galexrt/srcds_controller:runner-latest")
	}
	if c.Docker.LocalTimeFile == "" {
		c.Docker.LocalTimeFile = "/etc/localtime"
	}
	if c.Docker.NamePrefix == "" {
		c.Docker.NamePrefix = "game-"
	}
	if c.Docker.TimezoneFile == "" {
		c.Docker.TimezoneFile = "/etc/timezone"
	}

	// General
	if c.General == nil {
		c.General = &General{}
	}
	if c.General.Umask == 0 {
		c.General.Umask = 7
	}

	// Server
	if c.Server == nil {
		return fmt.Errorf("no server config found")
	}
	if c.Server.ACL == nil {
		c.Server.ACL = &ACL{
			Users:  []int{0},
			Groups: []int{0},
		}
	}
	if c.Server.Address == "" {
		return fmt.Errorf("no server address given")
	}
	if c.Server.Command == "" {
		c.Server.Command = "./srcds_run"
	}
	if c.Server.MapSelection == nil {
		c.Server.MapSelection = &MapSelection{
			Enabled: false,
		}
	}
	if c.Server.RCON != nil {
		if c.Server.RCON.Password == "" {
			return fmt.Errorf("no RCON password set")
		}
	}
	if c.Server.Port == 0 {
		return fmt.Errorf("no server port given")
	}

	if c.Server.SteamCMDDir == "" {
		// Get current user's home dir
		home, err := homedir.Dir()
		if err != nil {
			return err
		}
		c.Server.SteamCMDDir = path.Join(home, "steamcmd")
	}

	return nil
}

// General general config options
type General struct {
	Umask int `yaml:"umask"`
}

// GlobalConfigPath default global config file path
const GlobalConfigPath = "/etc/srcds_controller/config.yaml"

// GlobalConfig global config file always read from `/etc/srcds_controller/config.yaml`
type GlobalConfig struct {
	General *General             `yaml:"general"`
	Docker  *Docker              `yaml:"docker"`
	Checker *Checker             `yaml:"checker"`
	Checks  map[string]CheckOpts `yaml:"checks"`
}
