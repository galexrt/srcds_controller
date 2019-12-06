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
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/acarl005/stripansi"
	"github.com/galexrt/srcds_controller/pkg/chcloser"
	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/gin-gonic/gin"
	"github.com/kr/pty"
	log "github.com/sirupsen/logrus"
	"go.uber.org/zap"
	yaml "gopkg.in/yaml.v2"
)

var (
	logger        *zap.SugaredLogger
	tty           *os.File
	out           io.Reader
	mutx          sync.Mutex
	onExitCommand string
	chancloser    = &chcloser.ChannelCloser{}
	envAuthKey    string
	serverName    string
)

func main() {
	_ = syscall.Umask(7)

	loggerProd, _ := zap.NewDevelopment()
	defer loggerProd.Sync()
	logger = loggerProd.Sugar()

	serverName = os.Getenv("SRCDS_SERVER_NAME")
	if serverName == "" {
		logger.Fatal("no server name given through env var")
	}
	envRunnerPort := os.Getenv("SRCDS_RUNNER_PORT")
	if envRunnerPort == "" {
		logger.Fatal("no runner port given through env var")
	}
	envAuthKey = os.Getenv("SRCDS_RUNNER_AUTH_KEY")
	if envAuthKey == "" {
		logger.Fatal("no runner auth key given through env var")
	}
	runnerPort, err := strconv.Atoi(envRunnerPort)
	if err != nil {
		logger.Fatal("runner port conversion from string to int failed")
	}

	onExitCommand := os.Getenv("SRCDS_RUNNER_ONEXIT_COMMAND")
	if onExitCommand == "" {
		logger.Warn("no / empty onExitCommand given through env var")
	}

	if len(os.Args) <= 1 {
		logger.Fatal("no args given")
	}

	listenAddress := fmt.Sprintf("127.0.0.1:%d", runnerPort)

	logger.Infof("starting srcds_runner on %s with following args: %+v", listenAddress, os.Args[1:])

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
	r.GET("/rconPwUpdate", rconPwUpdate)
	r.POST("/rconPwUpdate", rconPwUpdate)

	go func() {
		if err = r.Run(listenAddress); err != nil {
			logger.Fatal(err)
			return
		}
	}()

	cmd := exec.CommandContext(ctx, os.Args[1], args...)
	cmd.Env = os.Environ()
	tty, err = pty.Start(cmd)
	if err != nil {
		logger.Fatal(err)
	}
	defer func() {
		if tty == nil {
			return
		}
		if err = tty.Close(); err != nil {
			//logger.Error(err)
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Info("beginning to stream logs")
		copyLogs(tty)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err = cmd.Wait(); err != nil {
			//logger.Errorf("error: %s\n", err)
		}

		chancloser.Close(stopCh)
		return
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

		mutx.Lock()

		if tty != nil {
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
	var authKey string
	if c.PostForm("auth-key") != "" {
		authKey = c.PostForm("auth-key")
	} else if c.Query("auth-key") != "" {
		authKey = c.Query("auth-key")
	} else {
		c.String(http.StatusBadRequest, "No auth key given.")
		return
	}
	if authKey != envAuthKey {
		c.String(http.StatusForbidden, "auth key not matching.")
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

	mutx.Lock()
	if _, err := tty.Write([]byte(command + "\n")); err != nil {
		c.String(http.StatusConflict, "error during command writing to server")
	}
	mutx.Unlock()
}

func rconPwUpdate(c *gin.Context) {
	var authKey string
	if c.PostForm("auth-key") != "" {
		authKey = c.PostForm("auth-key")
	} else if c.Query("auth-key") != "" {
		authKey = c.Query("auth-key")
	} else {
		c.String(http.StatusBadRequest, "No auth key given.")
		return
	}
	if authKey != envAuthKey {
		c.String(http.StatusForbidden, "auth key not matching.")
		return
	}

	var password string
	if c.PostForm("password") != "" {
		password = c.PostForm("password")
	} else if c.Query("password") == "" {
		password = c.Query("password")
	} else {
		c.String(http.StatusBadRequest, "No password given.")
		return
	}

	if tty == nil {
		c.String(http.StatusInternalServerError, "cmd tty is nil")
		return
	}
	mutx.Lock()
	if _, err := tty.Write([]byte(fmt.Sprintf("rcon_password %s\n", password))); err != nil {
		mutx.Unlock()
		c.String(http.StatusConflict, "error during command writing for rcon pw update to server")
		return
	}
	mutx.Unlock()
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

// reconciliation loop runs every 5 minutes to keep the RCON password in sync
func reconciliation(stopCh chan struct{}) {
	for {
		config.Cfg = &config.Config{}

		out, err := ioutil.ReadFile("/config/config.yaml")
		if err != nil {
			log.Fatal(err)
		}
		if err = yaml.Unmarshal(out, config.Cfg); err != nil {
			log.Fatal(err)
		}
		if err = config.Cfg.Verify(); err != nil {
			log.Fatal(err)
		}

		_, serverCfg := config.Cfg.Servers.GetByName(serverName)
		if serverCfg == nil {
			log.Errorf("no config for server %s found in config file", serverName)
		} else {
			mutx.Lock()
			if _, err := tty.Write([]byte(fmt.Sprintf("rcon_password %s\n\n", serverCfg.RCON.Password))); err != nil {
				mutx.Unlock()
				log.Errorf("error during command writing for rcon pw update to server")
				return
			}
			mutx.Unlock()
		}
		select {
		case <-time.After(5 * time.Minute):
		case <-stopCh:
			return
		}
	}
}
