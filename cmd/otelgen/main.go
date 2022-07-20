package main

import (
	"fmt"
	"log"
	"os"

	"github.com/krzko/otelgen/internal/cli"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	v := fmt.Sprintf("v%v-%v", version, commit)
	app := cli.New(v)

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
