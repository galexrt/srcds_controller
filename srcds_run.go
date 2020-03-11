package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	for {
		reader := bufio.NewReader(os.Stdin)
		text, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("ERROR: %+v\n", err)
			return
		}
		if strings.Contains(text, "srcds_controller_check") {
			fmt.Fprintf(os.Stderr, `Unknown command "srcds_controller_check"`+"\n")
			continue
		}
		fmt.Fprintf(os.Stderr, "%+v - %+v\n", time.Now(), text)
	}
}
