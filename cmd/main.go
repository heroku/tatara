package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	"github.com/heroku/tatara/cli"
)

func main() {
	os.Exit(runApp())
}

func runApp() int {
	exitChan := make(chan struct{})
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT)
	signal.Notify(signalChan, syscall.SIGTERM)
	go func() {
		<-signalChan
		close(exitChan)
	}()

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
		Exit:        exitChan,

		Commands: []cli.Command{
			cmdBuild,
			cmdRun,
			cmdExport,
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
