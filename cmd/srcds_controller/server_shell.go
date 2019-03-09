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
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/galexrt/srcds_controller/pkg/server"
	"github.com/spf13/cobra"
)

const helpText = `Following commands are available:
* list - Show list of available servers.
* logs [SERVER_NAME] - Show server logs.
* restart [SERVER_NAME] - Restart the server with the given name.
* start [SERVER_NAME] - Start the server with the given name.
* stop [SERVER_NAME] - Stop the server with the given name.
* command [SERVER_NAME] [COMMAND ...] - Run the given command in the console of the server with the given name.
* help - Shows available commands.

`

// serverShellCmd srcds_controller server shell.
var serverShellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Shell",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		fmt.Print(`Welcome to srcds_controller!
---
` + helpText)
		fmt.Print(`> $ `)

		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			input := scanner.Text()
			parts := strings.Split(input, " ")
			switch parts[0] {
			case "list":
				if err := server.List(cmd, parts[1:]); err != nil {
					fmt.Println(err)
				}
			case "logs":
				if len(parts) < 2 {
					fmt.Println("No server given.")
					break
				}
				if body, err := server.Logs(cmd, parts[1:]); err != nil {
					fmt.Println(err)
				} else {
					go func() {
						<-c
						body.Close()
					}()
					scanner := bufio.NewScanner(body)
					for scanner.Scan() {
						fmt.Println(scanner.Text())
					}
				}
			case "restart":
				if len(parts) < 2 {
					fmt.Println("No server given.")
					break
				}
				if err := server.Restart(cmd, parts[1:]); err != nil {
					fmt.Println(err)
				}
			case "start":
				if len(parts) < 2 {
					fmt.Println("No server given.")
					break
				}
				if err := server.Start(cmd, parts[1:]); err != nil {
					fmt.Println(err)
				}
			case "stop":
				if len(parts) < 2 {
					fmt.Println("No server given.")
					break
				}
				if err := server.Stop(cmd, parts[1:]); err != nil {
					fmt.Println(err)
				}
			case "remove":
				fallthrough
			case "rm":
				if len(parts) < 2 {
					fmt.Println("No server given.")
					break
				}
				if err := server.Remove(cmd, parts[1:]); err != nil {
					fmt.Println(err)
				}
			case "cmd":
				fallthrough
			case "command":
				if len(parts) < 2 {
					fmt.Println("No server given.")
					break
				}
				if len(parts) < 3 {
					fmt.Println("No command given.")
					break
				}
				serverName := parts[1]
				if err := server.SendCommand(serverName, parts[2:]); err != nil {
					fmt.Println(err)
				}
			case "bash":
				binary, lookErr := exec.LookPath("bash")
				if lookErr != nil {
					panic(lookErr)
				}
				args := []string{"bash"}
				env := os.Environ()
				if err := syscall.Exec(binary, args, env); err != nil {
					panic(err)
				}
			case "logout":
				fallthrough
			case "quit":
				fallthrough
			case "exit":
				fmt.Println("Exiting ...")
				return nil
			case "help":
				fmt.Print(helpText)
			default:
				fmt.Println("No command given.")
				fmt.Print(helpText)
			}

			fmt.Print("> $ ")
		}
		return nil
	},
}

func init() {
	serverCmd.AddCommand(serverShellCmd)
}
