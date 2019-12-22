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

package main

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/docker/docker/client"
	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/server"
	homedir "github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

const (
	// AllServers key to get all servers
	AllServers = "all"
)

var (
	cfgFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "sc",
	Short: "Client tool to manage gameservers run using srcds_controller project.",
}

func main() {
	Execute()
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
		DisableColors:    false,
		FullTimestamp:    false,
		TimestampFormat:  "2006-01-02 15:04:05",
	})
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.srcds_controller.yaml)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile == "" {
		// Get current work
		home, err := homedir.Dir()
		if err != nil {
			log.Fatal(err)
		}
		cfgFile = path.Join(home, ".srcds_controller.yaml")
	}

	config.Cfg = &config.Config{}
	config.FilePath = cfgFile

	if _, err := os.Stat(cfgFile); err == nil {
		out, err := ioutil.ReadFile(cfgFile)
		if err != nil {
			log.Fatal(err)
		}
		if err = yaml.Unmarshal(out, config.Cfg); err != nil {
			log.Fatal(err)
		}
		if err = config.Cfg.Verify(); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Fatal("no config found in home dir nor specified by flag")
	}
}

func init() {
	rootCmd.PersistentFlags().BoolP(AllServers, "a", false, "If all servers should be used")
	viper.BindPFlag(AllServers, rootCmd.PersistentFlags().Lookup(AllServers))
}

func initDockerCli(cmd *cobra.Command, args []string) error {
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}
	server.DockerCli = cli
	return err
}
