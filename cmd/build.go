package main

import (
	"fmt"
	"path/filepath"
	"errors"
	"context"
	"bytes"
	"archive/tar"
	"io/ioutil"

	"github.com/heroku/tatara/heroku"
	"github.com/sclevine/forge"
	"github.com/sclevine/forge/app"
	"github.com/sclevine/forge/engine"
	"github.com/sclevine/forge/engine/docker"
	"github.com/fatih/color"
	"github.com/heroku/tatara/cli"
	"github.com/heroku/tatara/fs"
	"github.com/heroku/tatara/ui"
	"github.com/docker/docker/api/types"
	dockerClient "github.com/docker/docker/client"
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


		engine, err := docker.New(&engine.EngineConfig{
			Exit: c.Exit,
		})
		if err != nil {
			return cli.ExitStatusUnknownError, err
		}
		defer engine.Close()

		stager := forge.NewStager(engine)

		if !c.Flags.Bool("skip-stack-pull") {
			err := ui.Loading("Downloading Build Image", engine.NewImage().Pull(stack))
			if err != nil {
				return cli.ExitStatusUnknownError, err
			}
		}

		herokuConfig, err := heroku.ReadConfig(appDir)
		if err == nil {
			if len(herokuConfig.Build.Buildpacks) > 0 {
				buildpacks = herokuConfig.ResolveBuildpacks()
			}

			runDockerfile := herokuConfig.ConstructDockerfile(RunStack)
			runImageName := fmt.Sprintf("%s:run", herokuConfig.Id)
			err = buildImage(runImageName, runDockerfile)
			if err != nil {
				return cli.ExitStatusUnknownError, err
			}

			buildDockerfile := herokuConfig.ConstructDockerfile(stack)
			buildImageName := fmt.Sprintf("%s:build", herokuConfig.Id)
			err = buildImage(buildImageName, buildDockerfile)
			if err != nil {
				return cli.ExitStatusUnknownError, err
			}

			stack = buildImageName
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
			StagingEnv: map[string]string{
				"STACK": stack,
			},
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

func buildImage(appName, dockerfile string) error {
	fmt.Println(fmt.Sprintf("Building %s", appName))

	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	dockerfileBytes := []byte(dockerfile)
	tarHeader := &tar.Header{
		Name: "Dockerfile",
		Size: int64(len(dockerfileBytes)),
	}
	err := tw.WriteHeader(tarHeader)
	if err != nil {
		return err
	}
	_, err = tw.Write(dockerfileBytes)
	if err != nil {
		return err
	}
	dockerFileTarReader := bytes.NewReader(buf.Bytes())

	buildOptions := types.ImageBuildOptions{
		Tags:       []string{appName},
		Dockerfile: "Dockerfile",
	}
	client, err := dockerClient.NewEnvClient()
	if err != nil {
		return err
	}

	buildResponse, err := client.ImageBuild(context.Background(), dockerFileTarReader, buildOptions)
	if err != nil {
		return fmt.Errorf("error starting build: %v", err)
	}
	response, err := ioutil.ReadAll(buildResponse.Body)
	if err != nil {
		return err
	}
	buildResponse.Body.Close()

	fmt.Println(string(response))
	return nil
}
