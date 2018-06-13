package main

import (
	"fmt"
	"path/filepath"
	"errors"
	"strconv"
	"os"
	"strings"

	"github.com/sclevine/forge"
	"github.com/sclevine/forge/engine"
	"github.com/sclevine/forge/engine/docker"
	"github.com/fatih/color"
	"github.com/heroku/tatara/cli"
	"github.com/heroku/tatara/fs"
	"github.com/heroku/tatara/ui"
	"github.com/heroku/tatara/heroku"
)

const (
	RunStack   = "packs/heroku-16:run"
)

var cmdRun = cli.Command{
	Name: "run",

	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "stack",
			Usage: "The name of the packs stack image to use",
		},
		cli.IntFlag{
			Name:  "port",
			Usage: "The local port to use",
		},
		cli.BoolFlag{
			Name:  "skip-stack-pull",
			Usage: "Use a local stack image only",
		},
		cli.StringSliceFlag{
			Name:  "env",
			Usage: "A single environment variable",
		},
	},

	Run: func(c *cli.Context) (int, error) {
		if len(c.Args) != 1 {
			fmt.Fprintln(c.App.UserErr, "required arguments: <app name>")
			return cli.ExitStatusInvalidArgs, errors.New("invalid arguments")
		}

		appName := filepath.Clean(c.Args[0])
		envVarsList := c.Flags.StringSlice("env")

		stack := c.Flags.String("stack")
		if stack == "" {
			stack = RunStack
		}

		envVars := make(map[string]string)
		for _, env := range envVarsList {
			parts := strings.SplitN(env, "=", 2)
			name := parts[0]
			value := parts[1]
			envVars[name] = value
		}

		port := c.Flags.Int("port")
		if port == 0 {
			port = 5000
		}

		sysFS := &fs.FS{}
		slugFile, slugSize, err := sysFS.ReadFile(fmt.Sprintf("./%s.slug", appName))
		if err != nil {
			return cli.ExitStatusInvalidArgs, err
		}
		slug := engine.NewStream(slugFile, slugSize)
		defer slug.Close()

		app := &forge.AppConfig{
			Name: appName,
			RunningEnv: envVars,
		}

		engine, err := docker.New(&engine.EngineConfig{
			Exit: c.Exit,
		})
		if err != nil {
			return cli.ExitStatusUnknownError, err
		}
		defer engine.Close()

		if !c.Flags.Bool("skip-stack-pull") {
			err = ui.Loading("Downloading Runtime Image", engine.NewImage().Pull(stack))
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
			fmt.Println(fmt.Sprintf("Using image: %s", imageName))
			stack = imageName
		}

		netConfig := &forge.NetworkConfig{
			HostIP:   "127.0.0.1",
			HostPort: strconv.FormatUint(uint64(port), 10),
			Port:     strconv.FormatUint(uint64(port), 10),
		}

		runner := forge.NewRunner(engine)
		runner.Logs = color.Output

		fmt.Println(fmt.Sprintf("Running %s on port %d...", appName, port))
		_, err = runner.Run(&forge.RunConfig{
			Droplet:       slug,
			Stack:         stack,
			Color:         color.GreenString,
			AppConfig:     app,
			NetworkConfig: netConfig,
			RootPath:      "/",
		})

		if err != nil {
			return cli.ExitStatusUnknownError, err
		}
		return cli.ExitStatusSuccess, nil
	},
}
