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

package userconfig

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/imdario/mergo"
	"gopkg.in/yaml.v2"
)

// Cfg variable holding the config object
var (
	Cfg *Config
)

// UserConfig user config pointing at a list of globs to server directories
type UserConfig struct {
	ServerDirectories []string `yaml:"serverDirectories"`
}

// Config actual config for the services
type Config struct {
	sync.Mutex
	Servers map[string]*config.Config
}

// Load load the configs into a Config object
func (uc *UserConfig) Load(globalCfg *config.GlobalConfig, cfgs *Config) error {
	configsToLoad := []string{}

	for _, fPath := range uc.ServerDirectories {
		fPath = path.Join(fPath, ".srcds_controller_server.yaml")
		matches, err := filepath.Glob(fPath)
		if err != nil {
			return err
		}
		configsToLoad = append(configsToLoad, matches...)
	}

	if len(configsToLoad) == 0 {
		return fmt.Errorf("no configs to load found in any serverDirectories path")
	}

	for _, confToLoad := range configsToLoad {
		if _, err := os.Stat(confToLoad); err == nil {
			out, err := ioutil.ReadFile(confToLoad)
			if err != nil {
				return err
			}
			serverCfg := &config.Config{}
			if err = yaml.Unmarshal(out, serverCfg); err != nil {
				return err
			}

			if serverCfg.Server.Name == "" {
				continue
			}

			serverCfg.Server.Path, _ = filepath.Split(confToLoad)

			if err := mergeGlobalWithServerCfg(globalCfg, serverCfg); err != nil {
				return err
			}

			if err = serverCfg.Verify(); err != nil {
				return err
			}

			cfgs.Servers[serverCfg.Server.Name] = serverCfg
		} else {
			return fmt.Errorf("skipping config %s due to error", confToLoad)
		}
	}

	return nil
}

func mergeGlobalWithServerCfg(globalCfg *config.GlobalConfig, cfg *config.Config) error {
	if cfg.General == nil {
		cfg.General = globalCfg.General
	} else if globalCfg.General != nil {
		if err := mergo.Merge(cfg.General, globalCfg.General); err != nil {
			return err
		}
	}

	if cfg.Docker == nil {
		cfg.Docker = globalCfg.Docker
	} else if globalCfg.Docker != nil {
		if err := mergo.Merge(cfg.Docker, globalCfg.Docker); err != nil {
			return err
		}
	}

	if cfg.Checker == nil {
		cfg.Checker = globalCfg.Checker
	} else if globalCfg.Checker != nil {
		if err := mergo.Merge(cfg.Checker, globalCfg.Checker); err != nil {
			return err
		}
	}

	if cfg.Checks == nil {
		cfg.Checks = globalCfg.Checks
	} else if globalCfg.Checks != nil {
		if err := mergo.Map(&cfg.Checks, globalCfg.Checks); err != nil {
			return err
		}
	}

	return nil
}
