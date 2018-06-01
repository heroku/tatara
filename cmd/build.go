package main

import (
	"fmt"
	"path/filepath"
	"errors"
	"os"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
	"github.com/heroku/tatara/heroku"
	"github.com/sclevine/forge"
	"github.com/sclevine/forge/app"
	"github.com/sclevine/forge/engine"
	"github.com/sclevine/forge/engine/docker"
	"github.com/fatih/color"
	"github.com/heroku/tatara/cli"
	"github.com/heroku/tatara/fs"
	"github.com/heroku/tatara/ui"
)

const (
	BuildStack   = "packs/heroku-16:build"
)

var cmdBuild = cli.Command{
	Name: "build",

	Flags: []cli.Flag{
		cli.StringSliceFlag{
			Name:  "buildpack",
			Usage: "A buildpack to use on this app",
		},
		cli.StringFlag{
			Name:  "stack",
			Usage: "The name of the packs stack image to use",
		},
		cli.BoolFlag{
			Name:  "skip-stack-pull",
			Usage: "Use a local stack image only",
		},
	},

	Run: func(c *cli.Context) (int, error) {
		if len(c.Args) != 2 {
			fmt.Fprintln(c.App.UserErr, "required arguments: <app directory> <app name>")
			return cli.ExitStatusInvalidArgs, errors.New("invalid arguments")
		}

		appDir := filepath.Clean(c.Args[0])
		appName := filepath.Clean(c.Args[1])
		buildpacks := c.Flags.StringSlice("buildpack")

		stack := c.Flags.String("stack")
		if stack == "" {
			stack = BuildStack
		}

		herokuYamlFile := filepath.Join(appDir, "heroku.yml")
		_, err := os.Stat(herokuYamlFile)
		if err == nil {
			configBytes, err := ioutil.ReadFile(herokuYamlFile)
			if err == nil {
				var herokuConfig heroku.Config
				yaml.Unmarshal(configBytes, &herokuConfig)
				rawBuildpacks := herokuConfig.Build.Buildpacks
				buildpacks = make([]string, len(rawBuildpacks))
				for i, buildpack := range rawBuildpacks {
					if strings.HasPrefix(buildpack, "https://") || strings.HasPrefix(buildpack, "http://"){
						buildpacks[i] = buildpack
					} else {
						buildpacks[i] = fmt.Sprintf("https://buildpack-registry.s3.amazonaws.com/buildpacks/%s.tgz", buildpack)
					}
				}
			}
		}

		engine, err := docker.New(&engine.EngineConfig{
			Exit: c.Exit,
		})
		if err != nil {
			return cli.ExitStatusUnknownError, err
		}
		defer engine.Close()

		stager := forge.NewStager(engine)

		if !c.Flags.Bool("skip-stack-pull") {
			err = ui.Loading("Downloading Build Image", engine.NewImage().Pull(stack))
			if err != nil {
				return cli.ExitStatusUnknownError, err
			}
		}

		slugPath := fmt.Sprintf("./%s.slug", appName)
		cachePath := fmt.Sprintf("./.%s.cache", appName)
		appTar, err := app.Tar(appDir, `^.+\.slug$`, `^\..+\.cache$`)
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
			Buildpacks: buildpacks,
		}

		slug, err := stager.Stage(&forge.StageConfig{
			AppTar:        appTar,
			Cache:         cache,
			CacheEmpty:    cacheSize == 0,
			BuildpackZips: nil,
			Stack:         stack,
			Color:         color.GreenString,
			AppConfig:     app,
			OutputPath:    "/out/slug.tgz",
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
