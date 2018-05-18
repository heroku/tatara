package main

import (
	"log"
	"os"
	"fmt"
	"runtime/pprof"

	"github.com/heroku/heroku-local-build/cli"
)

func main() {
	os.Exit(runApp())
}

func runApp() int {
	if os.Getenv("CPU_PROFILE") != "" {
		f, err := os.Create(os.Getenv("CPU_PROFILE"))
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	app := &cli.App{
		UserOut:     os.Stdout,
		UserErr:     os.Stderr,
		InternalOut: os.Stderr,

		Commands: []cli.Command{
			cmdBuild,
			cmdRun,
		},

		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "verbose",
				Usage: "verbose logging",
			},
		},
	}

	exitStatus, err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(app.InternalOut, err.Error())
	}

	return exitStatus
}
