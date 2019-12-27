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
	"sync"
)

// Cfg variable holding the global config object
var Cfg *Config

// Config config file struct
type Config struct {
	sync.RWMutex
	General *General             `yaml:"general"`
	Docker  *Docker              `yaml:"docker"`
	Server  *Server              `yaml:"server"`
	Checker *Checker             `yaml:"checker"`
	Checks  map[string]CheckOpts `yaml:"checks"`
}

// Verify verify the config file
func (c *Config) Verify() error {
	if c.General == nil {
		c.General = &General{
			Umask: 7,
		}
	}

	return nil
}

// General general config options
type General struct {
	Umask int `yaml:"umask"`
}
