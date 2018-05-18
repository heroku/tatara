package main

import (
	"fmt"
	"path/filepath"
	"errors"

	"github.com/sclevine/forge"
	"github.com/sclevine/forge/app"
	"github.com/sclevine/forge/engine"
	"github.com/sclevine/forge/engine/docker"
	"github.com/fatih/color"
	"github.com/heroku/heroku-local-build/cli"
	"github.com/heroku/heroku-local-build/fs"
)

const (
	BuildStack   = "packs/heroku-16:build"
)

var cmdBuild = cli.Command{
	Name: "build",

	Run: func(c *cli.Context) (int, error) {
		if len(c.Args) != 2 {
			fmt.Fprint(c.App.UserErr, "required arguments: <app directory> <app name>")
			return cli.ExitStatusInvalidArgs, errors.New("invalid arguments")
		}

		appDir := filepath.Clean(c.Args[0])
		appName := filepath.Clean(c.Args[1])

		// TODO add options for:
		//  - BuildStack name
		//  - buildpack URLs/names
		//  - SkipStackPull (default false)

		engine, err := docker.New(&engine.EngineConfig{})
		if err != nil {
			return cli.ExitStatusUnknownError, err
		}
		defer engine.Close()

		stager := forge.NewStager(engine)

		slugPath := fmt.Sprintf("./%s.slug", appName)
		cachePath := fmt.Sprintf("./.%s.cache", appName)

		appTar, err := app.Tar(appDir)
		if err != nil {
			return cli.ExitStatusUnknownError, err
		}
		defer appTar.Close()

		sysFS := &fs.FS{}
		cache, cacheSize, err := sysFS.OpenFile(cachePath)
		if err != nil {
			return cli.ExitStatusUnknownError, err
		}
		defer cache.Close()

		var app = &forge.AppConfig{
			Name: appName,
			Buildpack: "",
		}

		slug, err := stager.Stage(&forge.StageConfig{
			AppTar:        appTar,
			Cache:         cache,
			CacheEmpty:    cacheSize == 0,
			BuildpackZips: nil,
			Stack:         BuildStack,
			ForceDetect:   true,
			Color:         color.GreenString,
			AppConfig:     app,
			OutputFile:    "slug.tgz",
			SkipStackPull: true,
		})
		if err != nil {
			return cli.ExitStatusUnknownError, err
		}
		defer slug.Close()

		if err := streamOut(*sysFS, slug, slugPath); err != nil {
			return cli.ExitStatusUnknownError, err
		}

		return cli.ExitStatusSuccess, nil
	},
}

func streamOut(fs fs.FS, stream engine.Stream, path string) error {
	file, err := fs.WriteFile(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return stream.Out(file)
}
