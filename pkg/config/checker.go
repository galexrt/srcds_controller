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
	"time"
)

// Check config for a check, see `pkg/checks/` for available checks
type Check struct {
	Limit Limit     `yaml:"limit"`
	Name  string    `yaml:"name"`
	Opts  CheckOpts `yaml:"opts"`
}

// CheckOpts options that can be set for a check
type CheckOpts map[string]string

// Limit config with limits and actions ot execute when the limits (after or count)
// have been reached
type Limit struct {
	After   time.Duration `yaml:"after"`
	Count   int64         `yaml:"count"`
	Actions []string      `yaml:"actions"`
}

// Checker config for the checker.Checker
type Checker struct {
	Interval time.Duration `yamk:"interval"`
	Splay    Splay         `yaml:"splay"`
}

// Splay time splay config
type Splay struct {
	Start int `yaml:"start"`
	End   int `yaml:"end"`
}
