// Package main is the entry point for the trazr-gen application.
package main

import (
	"log"
	"os"

	"github.com/medxops/trazr-gen/internal/cli"
)

var (
	version string
	commit  string
	date    string
)

func main() {
	app := cli.New(version, commit, date)

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
