/*
Copyright 2018 Alexander Trost <galexrt@googlemail.com>

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
)

// Servers list of Server
type Servers []Server

// Server config/info for a server
type Server struct {
	Name       string  `yaml:"name"`
	Address    string  `yaml:"address"`
	Port       int     `yaml:"port"`
	ScreenName string  `yaml:"screenName"`
	Path       string  `yaml:"path"`
	Checks     []Check `yaml:"checks"`
}

// GetByName return server from list by name
func (s Servers) GetByName(name string) *Server {
	name = strings.ToLower(name)
	for _, server := range s {
		if strings.ToLower(server.Name) == name {
			return &server
		}
	}
	return nil
}
