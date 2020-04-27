/*
Copyright 2020 Alexander Trost <galexrt@googlemail.com>

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
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/acarl005/stripansi"
	"github.com/docker/docker/api/types/strslice"
	"github.com/fsnotify/fsnotify"
	"github.com/galexrt/srcds_controller/pkg/chcloser"
	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/google/gops/agent"
	"github.com/kr/pty"
	"github.com/prometheus/common/version"
	"go.uber.org/zap"
	yaml "gopkg.in/yaml.v2"
)

const (
	// ListenAddress server listen address
	ListenAddress = ".srcds_runner.sock"
	// ConfigFileName name of the config file for a server
	ConfigFileName = ".srcds_controller_server.yaml"
)

var (
	logger       *zap.SugaredLogger
	cfgMutex     = &sync.Mutex{}
	chancloser   = &chcloser.ChannelCloser{}
	tty          *os.File
	out          io.Reader
	consoleMutex sync.Mutex
)

func init() {
	rand.Seed(time.Now().Unix())
}

func getRandomMap(filter string) (string, error) {
	matches, err := filepath.Glob(filter)
	if err != nil {
		return "", err
	}

	// Example: rp_townsend_v2.bsp
	// the .bsp must be removed
	mapName := filepath.Base(matches[rand.Intn(len(matches))])
	mapName = strings.TrimSuffix(mapName, filepath.Ext(mapName))

	return mapName, nil
}

func setupServerArgs() []string {
	cfgMutex.Lock()
	defer cfgMutex.Unlock()

	syscall.Umask(config.Cfg.General.Umask)

	var err error
	chosenMap := config.Cfg.Server.MapSelection.FallbackMap
	if config.Cfg.Server.MapSelection.Enabled {
		chosenMap, err = getRandomMap(config.Cfg.Server.MapSelection.FileFilter)
		if err != nil {
			logger.Errorf(
				"failed to get a random map (filter: '%s'), using fallback %s. %+v",
				config.Cfg.Server.MapSelection.FallbackMap,
				config.Cfg.Server.MapSelection.FileFilter,
				err,
			)
		}
	}

	contArgs := strslice.StrSlice{
		config.Cfg.Server.Command,
		"-port",
		strconv.Itoa(config.Cfg.Server.Port),
	}

	for _, arg := range config.Cfg.Server.Flags {
		arg = strings.Replace(arg, "%RCON_PASSWORD%", config.Cfg.Server.RCON.Password, -1)
		arg = strings.Replace(arg, "%MAP_RANDOM%", chosenMap, -1)
		contArgs = append(contArgs, arg)
	}

	return contArgs
}

func main() {
	// Enable gops agent for troubleshooting
	if err := agent.Listen(agent.Options{
		ShutdownCleanup: true,
		ConfigDir:       "/tmp/agent",
	}); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	loggerProd, err := zap.NewDevelopment()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer loggerProd.Sync()
	logger = loggerProd.Sugar()

	logger.Infof("starting srcds_runner %s", version.Info())
	logger.Infof("build context %s", version.BuildContext())

	cfg, err := loadConfig()
	if err != nil {
		logger.Fatal(err)
	}
	if err := cfg.Verify(); err != nil {
		logger.Fatal(err)
	}
	cfgMutex.Lock()
	defer cfgMutex.Unlock()
	config.Cfg = cfg

	contArgs := setupServerArgs()
	logger.Infof("starting gameserver with cmd and args: %+v", contArgs)

	sigs := make(chan os.Signal, 1)
	stopCh := make(chan struct{})

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	wg := sync.WaitGroup{}

	// HTTP server config and run
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = ioutil.Discard

	r := gin.New()
	pprof.Register(r)
	r.Use(gin.Recovery())
	r.GET("/", cmdExecute)
	r.POST("/", cmdExecute)
	go listenAndServe(r)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := exec.CommandContext(ctx, contArgs[0], contArgs[1:]...)
	cmd.Env = os.Environ()
	tty, err = pty.Start(cmd)
	if err != nil {
		logger.Fatal(err)
	}

	wg.Add(2)
	go func() {
		defer wg.Done()
		logger.Info("beginning to stream logs from console")
		// copyLogs "automatically" returns when the tty has been closed
		// and all output has been processed
		copyLogs(tty)
	}()
	go func() {
		defer wg.Done()
		configWatchAndReconcile(stopCh)
	}()

	logger.Info("waiting for signals")
	<-sigs
	close(stopCh)

	consoleMutex.Lock()

	if tty != nil {
		cfgMutex.Lock()
		onExitCommand := config.Cfg.Server.OnExitCommand
		cfgMutex.Unlock()
		if onExitCommand != "" {
			logger.Infof("trying to run onExitCommand '%s'\n", onExitCommand)

			tty.Write([]byte("\n" + onExitCommand + "\n"))
			time.Sleep(5 * time.Second)
		}
	}

	time.Sleep(500 * time.Millisecond)

	if cmd.Process != nil {
		cmd.Process.Signal(syscall.SIGTERM)
	}

	cancel()

	if tty != nil {
		tty.Close()
	}

	logger.Info("waiting for everything to exit")
	wg.Wait()
	logger.Info("exiting srcds_runner")
}

func cmdExecute(c *gin.Context) {
	ok, err := checkACL(c.Request)
	if err != nil {
		c.String(http.StatusForbidden, fmt.Sprintf("permission denied. %+v", err))
		return
	}
	if !ok {
		c.String(http.StatusForbidden, "You don't have access to this server")
		return
	}

	var command string
	if c.PostForm("command") != "" {
		command = c.PostForm("command")
	} else if c.Query("command") != "" {
		command = c.Query("command")
	} else {
		c.String(http.StatusBadRequest, "No command given.")
		return
	}

	if tty == nil {
		c.String(http.StatusInternalServerError, "cmd tty is nil")
		return
	}

	consoleMutex.Lock()
	defer consoleMutex.Unlock()
	if _, err := tty.Write([]byte(command + "\n")); err != nil {
		c.String(http.StatusConflict, "error during command writing to server")
	}
}

func copyLogs(r io.Reader) error {
	buf := make([]byte, 512)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			outLine := stripansi.Strip(
				string(buf[0:n]),
			)

			outLine = cleanOutput(outLine)

			if lineToStderr(outLine) {
				os.Stderr.Write([]byte(
					outLine,
				))
			} else {
				os.Stdout.Write([]byte(
					outLine,
				))
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func loadConfig() (*config.Config, error) {
	out, err := ioutil.ReadFile(ConfigFileName)
	if err != nil {
		return nil, err
	}

	var cfg *config.Config
	if err := yaml.Unmarshal(out, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// configWatchAndReconcile loop runs every 5 minutes to keep the RCON password in sync
func configWatchAndReconcile(stopCh chan struct{}) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Errorf("error creating fsnotify watcher for config. %w", err)
		return
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Op&fsnotify.Write == fsnotify.Write {
					checkIfConfigChanged()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logger.Errorf("error during config fsnotify. %w", err)
			}
		}
	}()

	if err = watcher.Add(ConfigFileName); err != nil {
		logger.Errorf("failed to add watch for config file. %w", err)
		return
	}

	select {
	case <-stopCh:
		return
	}
}

func checkIfConfigChanged() {
	newCfg, err := loadConfig()
	if err != nil {
		logger.Errorf("failed to reload config. %w", err)
	}
	if err := config.Cfg.Verify(); err != nil {
		logger.Errorf("failed to verify reloaded config. %w", err)
	}

	if config.Cfg.Server.RCON.Password != newCfg.Server.RCON.Password {
		consoleMutex.Lock()
		defer consoleMutex.Unlock()
		if _, err := tty.Write([]byte(fmt.Sprintf("rcon_password %s\n\n", newCfg.Server.RCON.Password))); err != nil {
			logger.Errorf("failed to write rcon_password command to server console. %+v", err)
		}
	}

	logger.Info("config file has been reloaded")
	cfgMutex.Lock()
	defer cfgMutex.Unlock()
	config.Cfg = newCfg
}

func cleanOutput(in string) string {
	if strings.HasPrefix(in, "rcon_password") {
		in = "rcon_password XXXXXXXXX"
	}
	return in
}

func lineToStderr(in string) bool {
	if strings.Contains(in, "srcds_controller_check") {
		return true
	}

	return false
}
