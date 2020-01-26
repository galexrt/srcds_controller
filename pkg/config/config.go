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
	Cachet  *Cachet              `yaml:"cachet"`
}

// Verify verify the config file
func (c *Config) Verify() error {
	if c.General == nil {
		c.General = &General{
			Umask: 7,
		}
	}

	if c.Server == nil {
		return fmt.Errorf("no server config found")
	}
	if c.Docker == nil {
		c.Docker = &Docker{
			Image:      "galexrt/srcds_controller:runner-latest",
			NamePrefix: "game-",
		}
	}

	if c.Server.ACL == nil {
		c.Server.ACL = &ACL{
			Users:  []int{},
			Groups: []int{},
		}
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
	Cachet  *Cachet              `yaml:"cachet"`
	Docker  *Docker              `yaml:"docker"`
	Checker *Checker             `yaml:"checker"`
	Checks  map[string]CheckOpts `yaml:"checks"`
}

// Cachet cachet integration config
type Cachet struct {
	URL    string `yaml:"url"`
	APIKey string `yaml:"apiKey"`
}
