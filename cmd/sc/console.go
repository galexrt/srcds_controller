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
	"os"
	"path"
	"strings"
	"time"

	"github.com/galexrt/srcds_controller/pkg/linehistory"
	"github.com/galexrt/srcds_controller/pkg/server"
	"github.com/marcusolsson/tui-go"
	homedir "github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serverConsoleCmd represents the logs command
var serverConsoleCmd = &cobra.Command{
	Use:               "console",
	Aliases:           []string{"t", "con"},
	Short:             "Show server logs and allow commands to be directly posted to one or more servers",
	PersistentPreRunE: initDockerCli,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.SetOutput(ioutil.Discard)

		servers, err := checkServers(cmd, args)
		if err != nil {
			return err
		}

		var consoleError error

		historyStore, histFile, err := history()
		if err != nil {
			return err
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
			command := e.Text()
			for _, serverCfg := range servers {
				if err := server.SendCommand(serverCfg, []string{command}); err != nil {
					history.Append(tui.NewHBox(
						tui.NewLabel("ERROR "),
						tui.NewLabel(err.Error()),
						tui.NewSpacer(),
					))
				}
			}
			historyStore.Add(command)
			if viper.GetBool("history") {
				go func(line string) {
					if histFile != nil {
						if _, err := histFile.WriteString(command + "\n"); err != nil {
							log.Fatal(err)
						}
					}
				}(command)
			}
			input.SetText("")
		})

		inputBox := tui.NewHBox(input)
		inputBox.SetBorder(true)
		inputBox.SetSizePolicy(tui.Expanding, tui.Maximum)

		console := tui.NewVBox(historyBox, inputBox)
		console.SetSizePolicy(tui.Expanding, tui.Expanding)

		root := tui.NewHBox(console)

		ui, err := tui.New(root)
		if err != nil {
			return err
		}
		ui.SetKeybinding("Esc", func() { ui.Quit() })
		ui.SetKeybinding("Ctrl+C", func() { ui.Quit() })
		ui.SetKeybinding("PgUp", func() {
			historyScroll.Scroll(0, -1)
		})
		ui.SetKeybinding("PgDn", func() {
			historyScroll.Scroll(0, 1)
		})
		ui.SetKeybinding("Up", func() {
			histRet, _ := historyStore.Older(input.Text())
			input.SetText(histRet)
		})
		ui.SetKeybinding("Down", func() {
			histRet, _ := historyStore.Newer(input.Text())
			input.SetText(histRet)
		})

		outChan := make(chan string)
		errors := make(chan error)

		for _, serverCfg := range servers {
			stdout, stderr, err := server.Logs(serverCfg, 0*time.Millisecond, 10, true)
			if err != nil {
				ui.Quit()
				return err
			}
			if stdout == nil || stderr == nil {
				ui.Quit()
				return fmt.Errorf("server.Logs returned nil body. something is wrong. %+v", err)
			}

			go func(serverName string) {
				scanner := bufio.NewScanner(stdout)
				for scanner.Scan() {
					msg := scanner.Text()
					if len(servers) > 1 {
						msg = fmt.Sprintf("%s: %s", serverName, msg)
					}
					outChan <- msg
				}
				if scanner.Err() != nil {
					errors <- scanner.Err()
					return
				}
			}(serverCfg.Server.Name)
			go func(serverName string) {
				scanner := bufio.NewScanner(stderr)
				for scanner.Scan() {
					msg := scanner.Text()
					//outChan <- fmt.Sprintf("TEST : %#X : TEST", msg)
					if len(servers) > 1 {
						msg = fmt.Sprintf("%s: %s", serverName, msg)
					}
					outChan <- msg
				}
				if scanner.Err() != nil {
					errors <- scanner.Err()
					return
				}
			}(serverCfg.Server.Name)
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
	serverConsoleCmd.PersistentFlags().Bool("history", true, "If history should be enabled")
	viper.BindPFlag("history", serverConsoleCmd.PersistentFlags().Lookup("history"))

	rootCmd.AddCommand(serverConsoleCmd)
}

func history() (*linehistory.History, *os.File, error) {
	// Get current home dir
	home, err := homedir.Dir()
	if err != nil {
		return nil, nil, err
	}

	histFilePath := path.Join(home, ".srcds_controller_history")
	histFile, err := os.OpenFile(histFilePath, os.O_RDONLY|os.O_CREATE, 0660)
	if err != nil {
		return nil, nil, err
	}

	out, err := ioutil.ReadAll(histFile)
	if err != nil {
		histFile.Close()
		return nil, nil, err
	}
	histFile.Close()

	parts := strings.Split(string(out), "\n")
	partsLen := len(parts)

	wantedHistoryLength := 51
	if partsLen >= wantedHistoryLength {
		histFile, err = os.OpenFile(histFilePath, os.O_WRONLY|os.O_TRUNC, 0660)
		if err != nil {
			return nil, nil, err
		}

		parts = parts[partsLen-wantedHistoryLength:]
		if _, err := histFile.WriteString(strings.Join(parts, "\n")); err != nil {
			histFile.Close()
			return nil, nil, err
		}
		histFile.Close()
	}
	histFile, err = os.OpenFile(histFilePath, os.O_WRONLY|os.O_APPEND, 0660)
	if err != nil {
		return nil, nil, err
	}
	defer histFile.Close()

	historyStore := linehistory.NewHistory()

	for _, line := range parts {
		if line == "" {
			continue
		}
		historyStore.Add(line)
	}
	return historyStore, histFile, nil
}
