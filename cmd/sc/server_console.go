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
	"time"

	"github.com/galexrt/srcds_controller/pkg/server"
	"github.com/marcusolsson/tui-go"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// serverConsoleCmd represents the logs command
var serverConsoleCmd = &cobra.Command{
	Use:               "console",
	Aliases:           []string{"t"},
	Short:             "Show server logs and allow commands to be directly posted to one server",
	PersistentPreRunE: initDockerCli,
	Args:              cobra.RangeArgs(1, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		log.SetOutput(ioutil.Discard)
		var consoleError error

		serverName := args[0]

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
			if err := server.SendCommand(serverName, []string{e.Text()}); err != nil {
				history.Append(tui.NewHBox(
					tui.NewLabel("ERROR"),
					tui.NewLabel(err.Error()),
					tui.NewSpacer(),
				))
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

		go func() {
			body, err := server.Logs(serverName, 0*time.Millisecond, 100)
			if err != nil {
				ui.Quit()
				return
			}
			if body == nil {
				consoleError = fmt.Errorf("server.Logs returned nil body. something is wrong")
				ui.Quit()
				return
			}

			scanner := bufio.NewScanner(body)
			for scanner.Scan() {
				history.Append(tui.NewLabel(scanner.Text()))
				ui.Repaint()
			}

			if err = scanner.Err(); err != nil {
				consoleError = err
				ui.Quit()
				return
			}
			return
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
