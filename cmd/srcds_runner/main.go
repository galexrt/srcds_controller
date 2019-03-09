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

	"github.com/galexrt/srcds_controller/pkg/chcloser"
	"github.com/gin-gonic/gin"
	"github.com/kr/pty"
	"go.uber.org/zap"
)

var (
	logger        *zap.SugaredLogger
	tty           *os.File
	out           io.Reader
	mutx          sync.Mutex
	onExitCommand string
	chancloser    = &chcloser.ChannelCloser{}
	envAuthKey    string
)

func main() {
	loggerProd, _ := zap.NewDevelopment()
	defer loggerProd.Sync()
	logger = loggerProd.Sugar()

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
	if envRunnerPort == "" {
		logger.Warn("no onExitCommand given through env var")
	}

	if len(os.Args) <= 1 {
		logger.Fatal("no args given")
	}

	listenAddress := fmt.Sprintf("127.0.0.1:%d", runnerPort)

	logger.Infof("starting srcds_runner on %s with following args: %+v\n", listenAddress, os.Args[1:])

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
	defer tty.Close()

	wg.Add(1)
	go func() {
		defer wg.Done()
		copyLogs(tty)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err = cmd.Wait(); err != nil {
			logger.Errorf("error: %s\n", err)
		}

		chancloser.Close(stopCh)
		return
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-stopCh:
		case <-sigs:
		}

		mutx.Lock()
		defer mutx.Unlock()

		if onExitCommand != "" {
			logger.Infof("trying to run onExitCommand '%s'\n", onExitCommand)
			if _, err := tty.Write([]byte(onExitCommand + "\n")); err != nil {
				logger.Errorf("failed to write onExitCommand to server tty. %+v", err)
			}

		}
		if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
			logger.Errorf("failed to send SIGTERM signal to server process. %+v", err)
		}

		time.Sleep(6 * time.Second)

		cancel()
	}()

	wg.Wait()
	logger.Info("exiting srcds_runner")
}

func cmdExecute(c *gin.Context) {
	if c.PostForm("auth-key") == "" && c.Query("auth-key") == "" {
		c.String(http.StatusBadRequest, "No auth key given.")
		return
	}
	var authKey string
	if c.PostForm("auth-key") != "" {
		authKey = c.PostForm("auth-key")
	} else {
		authKey = c.Query("auth-key")
	}
	if authKey != envAuthKey {
		c.String(http.StatusForbidden, "auth key not matching.")
		return
	}

	if c.PostForm("command") == "" && c.Query("command") == "" {
		c.String(http.StatusBadRequest, "No command given.")
		return
	}
	var command string
	if c.PostForm("command") != "" {
		command = c.PostForm("command")
	} else {
		command = c.Query("command")
	}

	mutx.Lock()
	if _, err := tty.Write([]byte(command + "\n")); err != nil {
		c.String(http.StatusConflict, "error during command writing to server")
	}
	mutx.Unlock()
}

func rconPwUpdate(c *gin.Context) {
	if c.PostForm("auth-key") == "" && c.Query("auth-key") == "" {
		c.String(http.StatusBadRequest, "No auth key given.")
		return
	}
	var authKey string
	if c.PostForm("auth-key") != "" {
		authKey = c.PostForm("auth-key")
	} else {
		authKey = c.Query("auth-key")
	}
	if authKey != envAuthKey {
		c.String(http.StatusForbidden, "auth key not matching.")
		return
	}

	if c.PostForm("password") == "" && c.Query("password") == "" {
		c.String(http.StatusBadRequest, "No password given.")
		return
	}
	var password string
	if c.PostForm("password") != "" {
		password = c.PostForm("password")
	} else {
		password = c.Query("password")
	}

	mutx.Lock()
	if _, err := tty.Write([]byte(fmt.Sprintf("rcon_password %s\n", password))); err != nil {
		c.String(http.StatusConflict, "error during command writing for rcon pw update to server")
		mutx.Unlock()
		return
	}
	mutx.Unlock()

	envAuthKey = password
}

func copyLogs(r io.Reader) {
	buf := make([]byte, 80)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			os.Stdout.Write(buf[0:n])
		}
		if err != nil {
			break
		}
	}
}
