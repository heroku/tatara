package main

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/fatih/color"
	"github.com/heroku/tatara/cli"
	"github.com/heroku/tatara/fs"
	"github.com/heroku/tatara/heroku"
	"github.com/heroku/tatara/ui"
	"github.com/sclevine/forge"
	"github.com/sclevine/forge/app"
	"github.com/sclevine/forge/engine"
	"github.com/sclevine/forge/engine/docker"
)

const (
	BuildStack = "packs/heroku-16:build"
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
		cli.StringSliceFlag{
			Name:  "env",
			Usage: "A single environment variable",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug logging",
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
		envVarsList := c.Flags.StringSlice("env")
		debug := c.Flags.Bool("debug")

		stack := c.Flags.String("stack")
		if stack == "" {
			stack = BuildStack
		}

		envVars := make(map[string]string)
		for _, env := range envVarsList {
			parts := strings.SplitN(env, "=", 2)
			name := parts[0]
			value := parts[1]
			envVars[name] = value
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

			options := buildImageOptions{
				Debug:   debug,
				Verbose: true,
			}

			runDockerfile := herokuConfig.ConstructDockerfile(RunStack)
			runImageName := fmt.Sprintf("%s:run", herokuConfig.Id)
			err = buildImageWithDockerfile(runImageName, runDockerfile, options)
			if err != nil {
				return cli.ExitStatusUnknownError, err
			}

			buildDockerfile := herokuConfig.ConstructDockerfile(stack)
			buildImageName := fmt.Sprintf("%s:build", herokuConfig.Id)
			err = buildImageWithDockerfile(buildImageName, buildDockerfile, options)
			if err != nil {
				return cli.ExitStatusUnknownError, err
			}

			stack = buildImageName
		}

		if len(envVars) > 0 {
			err = applyEnvVars(stack, appName, envVars, debug)
			if err != nil {
				return cli.ExitStatusUnknownError, err
			}
			defer cleanUpEnvVarLayer(appName)
			stack = appName
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
			Name:       appName,
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

func buildImageWithDockerfile(appName, dockerfile string, options buildImageOptions) error {
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

	return buildImage(appName, buf, options)
}

type buildImageOptions struct {
	Debug   bool
	Verbose bool
}

func buildImage(appName string, dockerContext *bytes.Buffer, options buildImageOptions) error {
	if options.Debug {
		fmt.Println(fmt.Sprintf("Building %s", appName))
	}

	dockerFileTarReader := bytes.NewReader(dockerContext.Bytes())

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

	if options.Verbose || options.Debug {
		err = jsonmessage.DisplayJSONMessagesStream(buildResponse.Body, os.Stdout, 0, false, nil)
		if err != nil {
			return err
		}
	} else {
		_, err := ioutil.ReadAll(buildResponse.Body)
		if err != nil {
			return err
		}
		buildResponse.Body.Close()
	}

	return nil
}

func createTar(src string) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}
		header.Name = strings.TrimPrefix(strings.Replace(path, src, "", -1), string(filepath.Separator))

		if !info.Mode().IsRegular() {
			return nil
		}

		err = tw.WriteHeader(header)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		defer file.Close()
		if err != nil {
			return err
		}

		_, err = io.Copy(tw, file)
		if err != nil {
			return err
		}

		return nil
	})

	return buf, err
}

func applyEnvVars(stack string, newStack string, env map[string]string, debug bool) error {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.Remove(tmpDir)

	envDir := fmt.Sprintf("%s/env", tmpDir)
	err = os.Mkdir(envDir, 0755)
	if err != nil {
		return err
	}

	for name, value := range env {
		filePath := fmt.Sprintf("%s/%s/%s", tmpDir, "env", name)
		ioutil.WriteFile(filePath, []byte(value), 0644)
	}
	dockerEnvDir := "/tmp/env"
	dockerfile := fmt.Sprintf(`FROM %s
COPY env %s
`, stack, dockerEnvDir)
	filePath := fmt.Sprintf("%s/Dockerfile", tmpDir)
	ioutil.WriteFile(filePath, []byte(dockerfile), 0644)

	tarball, err := createTar(tmpDir)
	if err != nil {
		return err
	}

	options := buildImageOptions{
		Debug:   debug,
		Verbose: false,
	}

	return buildImage(newStack, tarball, options)
}

func cleanUpEnvVarLayer(stack string) error {
	client, err := dockerClient.NewEnvClient()
	if err != nil {
		fmt.Printf("Couldn't remove Env Var layer: %s", err.Error())
		return err
	}

	removeOptions := types.ImageRemoveOptions{
		Force: true,
	}

	_, err = client.ImageRemove(context.Background(), stack, removeOptions)
	if err != nil {
		fmt.Printf("Couldn't remove Env Var layer: %s", err.Error())
		return err
	}

	return nil
}
