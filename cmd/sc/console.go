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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
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
		if !viper.GetBool("debug") {
			log.SetOutput(ioutil.Discard)
		}

		servers, err := checkServers(cmd, args)
		if err != nil {
			return err
		}

		// Init console history system
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

		outChan := make(chan string)

		input.OnSubmit(func(e *tui.Entry) {
			command := e.Text()
			if command == "" {
				return
			}
			for _, srvCfg := range servers {
				if err := server.SendCommand(srvCfg, []string{command}); err != nil {
					outChan <- fmt.Sprintf("CONSOLE ERROR: %+v", strings.Trim(err.Error(), "\n"))
				}
			}
			historyStore.Add(command)
			if viper.GetBool("history") {
				if histFile != nil {
					if _, err := histFile.WriteString(command + "\n"); err != nil {
						log.Error(err)
					}
				}
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

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ui.SetKeybinding("Esc", func() { cancel() })
		ui.SetKeybinding("Ctrl+C", func() { cancel() })
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

		wg := &sync.WaitGroup{}

		for _, srvCfg := range servers {
			cmd, stdout, stderr, err := server.Logs(ctx, srvCfg, 0*time.Millisecond, 10, true)
			if err != nil {
				return err
			}
			if stdout == nil || stderr == nil {
				return fmt.Errorf("unable to get server container logs. server.Logs returned nil body. %+v", err)
			}

			wg.Add(3)
			go func(serverName string, stream io.ReadCloser) {
				defer wg.Done()
				defer stream.Close()
				scanner := bufio.NewScanner(stream)
				for scanner.Scan() {
					msg := scanner.Text()
					if len(servers) > 1 {
						msg = fmt.Sprintf("%s: %s", serverName, msg)
					}
					outChan <- msg
				}
				if scanner.Err() != nil {
					log.Error(scanner.Err())
					return
				}
			}(srvCfg.Server.Name, stdout)
			go func(serverName string, stream io.ReadCloser) {
				defer wg.Done()
				defer stream.Close()
				scanner := bufio.NewScanner(stream)
				for scanner.Scan() {
					msg := scanner.Text()
					if len(servers) > 1 {
						msg = fmt.Sprintf("%s: %s", serverName, msg)
					}
					if viper.GetBool("debug") {
						outChan <- msg
					}
				}
				if scanner.Err() != nil {
					log.Error(scanner.Err())
					return
				}
			}(srvCfg.Server.Name, stderr)
			go func(c *exec.Cmd) {
				defer wg.Done()
				c.Wait()
			}(cmd)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					ui.Quit()
					return
				case out := <-outChan:
					history.Append(tui.NewLabel(out))
				}
			}
		}()
		time.Sleep(100 * time.Millisecond)

		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(viper.GetDuration("repaint-interval")):
					ui.Repaint()
				}
			}
		}()

		if err := ui.Run(); err != nil {
			log.Error(err)
		}

		wg.Wait()
		return nil
	},
}

func init() {
	serverConsoleCmd.PersistentFlags().Bool("history", true, "If history should be enabled")
	serverConsoleCmd.PersistentFlags().Duration("repaint-interval", 250*time.Millisecond, "Console repaint interval, do not change unless you know what you are doing!")
	viper.BindPFlag("history", serverConsoleCmd.PersistentFlags().Lookup("history"))
	viper.BindPFlag("repaint-interval", serverConsoleCmd.PersistentFlags().Lookup("repaint-interval"))

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

	historyStore := linehistory.NewHistory()

	for _, line := range parts {
		if line == "" {
			continue
		}
		historyStore.Add(line)
	}
	return historyStore, histFile, nil
}
