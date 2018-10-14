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
	"time"
)

var Cfg *Config

type Config struct {
	Servers []Server             `yaml:"servers"`
	Checker Checker              `yaml:"checker"`
	Checks  map[string]CheckOpts `yaml:"checks"`
}

type Check struct {
	Limit Limit     `yaml:"limit"`
	Name  string    `yaml:"name"`
	Opts  CheckOpts `yaml:"opts"`
}

type Limit struct {
	After   time.Duration `yaml:"after"`
	Count   int64         `yaml:"count"`
	Actions []string      `yaml:"actions"`
}

type CheckOpts map[string]string

type Server struct {
	Name       string  `yaml:"name"`
	Address    string  `yaml:"address"`
	Port       int     `yaml:"port"`
	ScreenName string  `yaml:"screenName"`
	Path       string  `yaml:"path"`
	Checks     []Check `yaml:"checks"`
}

type Checker struct {
	Interval time.Duration `yamk:"interval"`
	Splay    Splay         `yaml:"splay"`
}

type Splay struct {
	Start int `yaml:"start"`
	End   int `yaml:"end"`
}
