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
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/galexrt/srcds_controller/pkg/chcloser"
	"github.com/gin-gonic/gin"
)

var (
	stdin         io.WriteCloser
	out           io.Reader
	mutx          sync.Mutex
	onExitCommand string
	chancloser    = &chcloser.ChannelCloser{}
	envAuthKey    string
)

func init() {
	flag.StringVar(&onExitCommand, "on-exit-cmd", "", "command to run on exit (e.g., signal received)")
}

func main() {
	flag.Parse()

	envRunnerID := os.Getenv("SRCDS_RUNNER_ID")
	if envRunnerID == "" {
		log.Fatal("no runner ID given through env var")
	}
	envAuthKey = os.Getenv("SRCDS_RUNNER_AUTH_KEY")
	if envAuthKey == "" {
		log.Fatal("no runner auth key given through env var")
	}
	runnerID, err := strconv.Atoi(envRunnerID)
	if err != nil {
		log.Fatal("runner ID conversion from string to int failed")
	}

	if len(os.Args) <= 1 {
		log.Fatal("no args given")
	}

	listenAddress := fmt.Sprintf("127.0.0.1:4%03d", runnerID)

	log.Printf("starting srcds_runner on %s with following args: %+v\n", listenAddress, os.Args[1:])

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
	if len(os.Args) >= 2 {
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
		r.Run(listenAddress)
	}()

	cmd := exec.CommandContext(ctx, os.Args[1], args...)

	stdin, err = cmd.StdinPipe()
	if err != nil {
		log.Printf("error: %s\n", err)
		os.Exit(1)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("error: %s\n", err)
		chancloser.Close(stopCh)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fmt.Println(err)
		chancloser.Close(stopCh)
		os.Exit(1)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(bufio.NewReader(stdout))
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
		if scanner.Err() != nil {
			log.Printf("error: %s\n", scanner.Err())
			chancloser.Close(stopCh)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()

		scanner := bufio.NewScanner(bufio.NewReader(stderr))
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
		if scanner.Err() != nil {
			log.Printf("error: %s\n", scanner.Err())
			chancloser.Close(stopCh)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := cmd.Start(); err != nil {
			fmt.Println(err)
			chancloser.Close(stopCh)
			return
		}

		if err := cmd.Wait(); err != nil {
			log.Printf("error: %s\n", err)
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

		if onExitCommand != "" {
			log.Printf("trying to run onExitCommand '%s'\n", onExitCommand)
			mutx.Lock()
			if _, err := stdin.Write([]byte(onExitCommand + "\n")); err != nil {
				log.Printf("error: %s\n", err)
			}
			mutx.Unlock()
		}

		cancel()
	}()

	wg.Wait()
	log.Println("exiting srcds_runner")
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
	if _, err := stdin.Write([]byte(command + "\n")); err != nil {
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
	if _, err := stdin.Write([]byte(fmt.Sprintf("rcon_password %s\n", password))); err != nil {
		c.String(http.StatusConflict, "error during command writing for rcon pw update to server")
		mutx.Unlock()
		return
	}
	mutx.Unlock()

	envAuthKey = password
}
