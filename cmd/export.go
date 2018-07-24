package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"os"

	"github.com/buildpack/forge"
	"github.com/buildpack/forge/engine"
	"github.com/buildpack/forge/engine/docker"
	"github.com/heroku/tatara/cli"
	"github.com/heroku/tatara/fs"
	"github.com/heroku/tatara/ui"
	"github.com/heroku/tatara/heroku"
)

var cmdExport = cli.Command{
	Name: "export",

	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "stack",
			Usage: "The name of the packs stack image to use",
		},
		cli.StringFlag{
			Name:  "tag",
			Usage: "Tag name to use for the docker image (defaults to app name)",
		},
		cli.BoolFlag{
			Name:  "skip-stack-pull",
			Usage: "Use a local stack image only",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug logging",
		},
	},

	Run: func(c *cli.Context) (int, error) {
		if len(c.Args) != 1 {
			fmt.Fprintln(c.App.UserErr, "required arguments: <app name>")
			return cli.ExitStatusInvalidArgs, errors.New("invalid arguments")
		}

		appName := filepath.Clean(c.Args[0])
		debug := c.Flags.Bool("debug")

		stack := c.Flags.String("stack")
		if stack == "" {
			stack = RunStack
		}

		tag := c.Flags.String("tag")
		if tag == "" {
			tag = appName
		}

		sysFS := &fs.FS{}
		slugFilename := fmt.Sprintf("./%s.slug", appName)
		slugFile, slugSize, err := sysFS.ReadFile(slugFilename)
		if err != nil {
			fmt.Fprintln(c.App.UserErr, fmt.Sprintf("Could not read slug file: %s", slugFilename))
			return cli.ExitStatusInvalidArgs, err
		}

		slug := engine.NewStream(slugFile, slugSize)
		defer slug.Close()

		engine, err := docker.New(&engine.EngineConfig{
			Exit: c.Exit,
		})
		if err != nil {
			return cli.ExitStatusUnknownError, err
		}
		defer engine.Close()

		if !c.Flags.Bool("skip-stack-pull") {
			err := ui.Loading("Downloading Runtime Image", engine.NewImage().Pull(stack))
			if err != nil {
				return cli.ExitStatusUnknownError, err
			}
		}

		curDir, err := os.Getwd()
		if err != nil {
			return cli.ExitStatusUnknownError, err
		}

		herokuConfig, err := heroku.ReadConfig(curDir)
		if err == nil {
			imageName := fmt.Sprintf("%s:run", herokuConfig.Id)
			if debug {
				fmt.Println(fmt.Sprintf("Using image: %s", imageName))
			}
			stack = imageName
		}

		exporter := forge.NewExporter(engine)

		id, err := exporter.Export(&forge.ExportConfig{
			Droplet:    slug,
			Stack:      stack,
			Ref:        tag,
			WorkingDir: "/app",
			OutputDir:  "/",
			AppConfig: &forge.AppConfig{
				Name:    appName,
			},
		})

		if err != nil {
			return cli.ExitStatusUnknownError, err
		}

		fmt.Fprintln(c.App.UserOut, fmt.Sprintf("Exported image %s with ID: %s", tag, id))

		return cli.ExitStatusSuccess, nil
	},
}
