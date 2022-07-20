package main

import (
	"log"
	"os"

	"github.com/krzko/otelgen/internal/cli"
)

var version string = "0.0.1"

func main() {
	app := cli.New(version)

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
