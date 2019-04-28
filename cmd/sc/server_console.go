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
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/galexrt/srcds_controller/pkg/config"
	"github.com/galexrt/srcds_controller/pkg/server"
	"github.com/marcusolsson/tui-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serverConsoleCmd represents the logs command
var serverConsoleCmd = &cobra.Command{
	Use:               "console",
	Aliases:           []string{"t"},
	Short:             "Show server logs and allow commands to be directly posted to one server",
	PersistentPreRunE: initDockerCli,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.SetOutput(ioutil.Discard)
		var consoleError error

		var servers []string
		if viper.GetBool(AllServers) || strings.ToLower(args[0]) == AllServers {
			for _, srv := range config.Cfg.Servers {
				servers = append(servers, srv.Name)
			}
		} else {
			servers = strings.Split(args[0], ",")
		}
		if len(servers) == 0 {
			return fmt.Errorf("no server(s) given, please provide a server list as the first argument, example: `sc " + cmd.Name() + " SERVER_A,SERVER_B` or `all` instead of the server list")
		}

		history := tui.NewVBox()

		historyScroll := tui.NewScrollArea(history)
		historyScroll.SetAutoscrollToBottom(true)

		historyBox := tui.NewVBox(historyScroll)
		historyBox.SetBorder(true)

		input := tui.NewEntry()
		input.SetFocused(true)
		input.SetSizePolicy(tui.Expanding, tui.Maximum)

		input.OnSubmit(func(e *tui.Entry) {
			if e.Text() == "" {
				return
			}
			for _, serverName := range servers {
				if err := server.SendCommand(serverName, []string{e.Text()}); err != nil {
					history.Append(tui.NewHBox(
						tui.NewLabel("ERROR"),
						tui.NewLabel(err.Error()),
						tui.NewSpacer(),
					))
				}
			}
			input.SetText("")
		})

		inputBox := tui.NewHBox(input)
		inputBox.SetBorder(true)
		inputBox.SetSizePolicy(tui.Expanding, tui.Maximum)

		chat := tui.NewVBox(historyBox, inputBox)
		chat.SetSizePolicy(tui.Expanding, tui.Expanding)

		root := tui.NewHBox(chat)

		ui, err := tui.New(root)
		if err != nil {
			return err
		}
		ui.SetKeybinding("Esc", func() { ui.Quit() })
		ui.SetKeybinding("Ctrl+C", func() { ui.Quit() })

		outChan := make(chan string)
		errors := make(chan error)

		for _, serverName := range servers {
			stdin, stderr, err := server.Logs(serverName, 0*time.Millisecond, 100)
			if err != nil {
				ui.Quit()
				return err
			}
			if stdin == nil || stderr == nil {
				ui.Quit()
				return fmt.Errorf("server.Logs returned nil body. something is wrong. %+v", err)
			}

			go func(serverName string) {
				scanner := bufio.NewScanner(stdin)
				for scanner.Scan() {
					outChan <- scanner.Text()
				}
				if scanner.Err() != nil {
					errors <- scanner.Err()
					return
				}
			}(serverName)
			go func(serverName string) {
				scanner := bufio.NewScanner(stderr)
				for scanner.Scan() {
					outChan <- scanner.Text()
				}
				if scanner.Err() != nil {
					errors <- scanner.Err()
					return
				}
			}(serverName)
		}

		go func() {
			for {
				select {
				case out := <-outChan:
					history.Append(tui.NewLabel(out))
					ui.Repaint()
				case erro := <-errors:
					consoleError = erro
					ui.Quit()
					return
				}
			}
		}()

		if err := ui.Run(); err != nil {
			return err
		}
		return consoleError
	},
}

func init() {
	rootCmd.AddCommand(serverConsoleCmd)
}
