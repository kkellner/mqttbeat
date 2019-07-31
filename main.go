package main

import (
	"github.com/tmechen/mqttbeat/cmd"
	"os"
)

func main() {
	err := cmd.RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
