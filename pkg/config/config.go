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
	"path/filepath"

	"github.com/coreos/pkg/capnslog"
)

var logger = capnslog.NewPackageLogger("github.com/galexrt/srcds_controller", "pkg/config")

// Cfg variable holding the global config object
var Cfg *Config

// FilePath path to config file
var FilePath string

// Config config file struct
type Config struct {
	General General              `yaml:"general`
	Docker  Docker               `yaml:"docker"`
	Servers Servers              `yaml:"servers"`
	Checker Checker              `yaml:"checker"`
	Checks  map[string]CheckOpts `yaml:"checks"`
}

// Verify verify the config file
func (c *Config) Verify() error {
	for k, server := range c.Servers {
		cleanedPath, err := filepath.Abs(server.Path)
		if err != nil {
			return err
		}
		if cleanedPath != server.Path {
			logger.Debugf("cleaned server %s path from %s to %s", server.Name, server.Path, cleanedPath)
			c.Servers[k].Path = cleanedPath
		}
	}
	return nil
}

// General general config options
type General struct {
	Umask string `yaml:"umask"`
}
