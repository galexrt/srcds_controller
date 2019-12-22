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
	"gopkg.in/yaml.v2"
)

// Cfg variable holding the config object
var Cfg *Config

// UserConfig
type UserConfig struct {
	ServerDirectories []string `yaml:"serverDirectories"`
}

type Config struct {
	sync.Mutex
	Servers map[string]*config.Config
}

func (uc *UserConfig) Load(cfgs *Config) error {
	configsToLoad := []string{}

	for _, fPath := range uc.ServerDirectories {
		fPath = path.Join(fPath, ".srcds_controller_server.yaml")
		matches, err := filepath.Glob(fPath)
		if err != nil {
			return err
		}
		configsToLoad = append(configsToLoad, matches...)
	}

	for _, confToLoad := range configsToLoad {
		if _, err := os.Stat(confToLoad); err == nil {
			out, err := ioutil.ReadFile(confToLoad)
			if err != nil {
				return err
			}
			cfg := &config.Config{}
			if err = yaml.Unmarshal(out, cfg); err != nil {
				return err
			}
			if err = cfg.Verify(); err != nil {
				return err
			}

			if cfg.Server.Name == "" {
				continue
			}

			cfg.Server.Path, _ = filepath.Split(confToLoad)

			cfgs.Servers[cfg.Server.Name] = cfg
		} else {
			return fmt.Errorf("skipping config %s due to error", confToLoad)
		}
	}

	return nil
}
