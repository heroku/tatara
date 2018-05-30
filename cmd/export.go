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
		slugFile, slugSize, err := sysFS.ReadFile(fmt.Sprintf("./%s.slug", appName))
		if err != nil {
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
		exporter := forge.NewExporter(engine)

		id, err := exporter.Export(&forge.ExportConfig{
			Droplet:   slug,
			Stack:     RunStack,
			Ref:       appName,
			AppConfig: &forge.AppConfig{},
		})

		if err != nil {
			return cli.ExitStatusUnknownError, err
		}

		fmt.Println("Exported as %s with ID: %s", appName, id)

		return cli.ExitStatusSuccess, nil
	},
}