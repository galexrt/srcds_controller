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
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/acarl005/stripansi"
	"github.com/galexrt/srcds_controller/pkg/chcloser"
	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/gin-gonic/gin"
	"github.com/kr/pty"
	"go.uber.org/zap"
	yaml "gopkg.in/yaml.v2"
)

const (
	// ListenAddress server listen address
	ListenAddress = "unix:///socket/socket.sock"
)

var (
	logger       *zap.SugaredLogger
	chancloser   = &chcloser.ChannelCloser{}
	tty          *os.File
	out          io.Reader
	consoleMutex sync.Mutex

	serverName string
)

func main() {
	loggerProd, err := zap.NewDevelopment()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer loggerProd.Sync()
	logger = loggerProd.Sugar()

	serverName = os.Getenv("SRCDS_CONTROLLER_SERVER_NAME")
	if serverName == "" {
		logger.Fatal("no server name given from env var")
	}

	config.Cfg = &config.Config{}
	loadConfig()
	if err := config.Cfg.Verify(); err != nil {
		logger.Fatal(err)
	}

	// Check if server config exists
	serverCfg := config.Cfg.Servers.GetByName(serverName)
	if serverCfg == nil {
		logger.Fatalf("server %s not found in config file", serverName)
	}

	syscall.Umask(config.Cfg.General.Umask)

	if len(os.Args) <= 1 {
		logger.Fatal("no server args given")
	}

	logger.Infof("starting srcds_runner on %s with following args: %+v", ListenAddress, os.Args[1:])

	sigs := make(chan os.Signal, 1)
	stopCh := make(chan struct{})

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	wg := sync.WaitGroup{}

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if !chancloser.IsClosed {
			cancel()
		}
	}()

	args := []string{}
	if len(os.Args) > 2 {
		args = os.Args[2:]
	}

	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = ioutil.Discard

	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/", cmdExecute)
	r.POST("/", cmdExecute)

	go func() {
		if err := r.Run(ListenAddress); err != nil {
			logger.Fatal(err)
			return
		}
	}()

	cmd := exec.CommandContext(ctx, os.Args[1], args...)
	cmd.Env = os.Environ()
	tty, err := pty.Start(cmd)
	if err != nil {
		logger.Fatal(err)
	}
	defer func() {
		if tty == nil {
			return
		}
		tty.Close()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("beginning to stream logs from tty console")
		copyLogs(tty)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		cmd.Wait()
		chancloser.Close(stopCh)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		reconciliation(stopCh)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-sigs:
		case <-stopCh:
		}

		consoleMutex.Lock()

		if tty != nil {
			config.Cfg.Lock()
			serverCfg := config.Cfg.Servers.GetByName(serverName)
			config.Cfg.Unlock()
			onExitCommand := serverCfg.OnExitCommand
			if onExitCommand != "" {
				logger.Infof("trying to run onExitCommand '%s'\n", onExitCommand)

				if _, err := tty.Write([]byte("\n\n" + onExitCommand + "\n")); err != nil {
					//logger.Errorf("failed to write onExitCommand to server tty. %+v", err)
				}
				time.Sleep(5 * time.Second)
			}
		}

		time.Sleep(500 * time.Millisecond)

		if cmd.Process != nil {
			if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
				//logger.Errorf("failed to send SIGTERM signal to server process. %+v", err)
			}
		}

		cancel()
	}()

	wg.Wait()
	logger.Info("exiting srcds_runner")
}

func cmdExecute(c *gin.Context) {
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
			os.Stdout.Write([]byte(
				stripansi.Strip(
					string(buf[0:n]),
				),
			),
			)
		}
		if err == io.EOF {
			//logger.Info("copyLogs: received EOF from given log source")
			return nil
		}
		if err != nil {
			//logger.Error(err)
			return err
		}
	}
}

func loadConfig() error {
	out, err := ioutil.ReadFile("/config/config.yaml")
	if err != nil {
		return err
	}
	config.Cfg.Lock()
	defer config.Cfg.Unlock()
	if err := yaml.Unmarshal(out, config.Cfg); err != nil {
		return err
	}
	return config.Cfg.Verify()
}

// reconciliation loop runs every 5 minutes to keep the RCON password in sync
func reconciliation(stopCh chan struct{}) {
	for {
		if err := loadConfig(); err != nil {
			logger.Fatal(err)
			return
		}
		config.Cfg.Lock()
		serverCfg := config.Cfg.Servers.GetByName(serverName)
		config.Cfg.Unlock()
		if serverCfg == nil {
			logger.Errorf("no config for server %s found in config file", serverName)
		} else {
			consoleMutex.Lock()
			if _, err := tty.Write([]byte(fmt.Sprintf("rcon_password %s\n\n", serverCfg.RCON.Password))); err != nil {
				consoleMutex.Unlock()
				logger.Errorf("failed to write rcon_password command to server console. %+v", err)
			}
			consoleMutex.Unlock()
		}
		select {
		case <-time.After(5 * time.Minute):
		case <-stopCh:
			return
		}
	}
}
