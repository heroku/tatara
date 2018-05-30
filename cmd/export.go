package main

import (
	"fmt"
	"path/filepath"
	"errors"

	"github.com/sclevine/forge/engine"
	"github.com/heroku/heroku-local-build/cli"
	"github.com/heroku/heroku-local-build/fs"
	"github.com/sclevine/forge"
	"github.com/sclevine/forge/engine/docker"
	"github.com/heroku/heroku-local-build/ui"
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
	},

	Run: func(c *cli.Context) (int, error) {
		if len(c.Args) != 1 {
			fmt.Fprintln(c.App.UserErr, "required arguments: <app name>")
			return cli.ExitStatusInvalidArgs, errors.New("invalid arguments")
		}

		appName := filepath.Clean(c.Args[0])

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

		exporter := forge.NewExporter(engine)

		id, err := exporter.Export(&forge.ExportConfig{
			Droplet:   slug,
			Stack:     RunStack,
			Ref:       tag,
			RootPath:  "/",
			HomePath:  "app",
			AppConfig: &forge.AppConfig{
				Name: appName,
			},
		})

		if err != nil {
			return cli.ExitStatusUnknownError, err
		}

		fmt.Fprintln(c.App.UserOut, fmt.Sprintf("Exported image %s with ID: %s", tag, id))

		return cli.ExitStatusSuccess, nil
	},
}