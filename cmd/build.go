package main

import (
	"fmt"
	"os"
	"strings"
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

    	// TODO put in setup (this will be reused in run)
		proxy := engine.ProxyConfig{
			HTTPProxy:  firstEnv("HTTP_PROXY", "http_proxy"),
			HTTPSProxy: firstEnv("HTTPS_PROXY", "https_proxy"),
			NoProxy:    firstEnv("NO_PROXY", "no_proxy"),
		}
		if useProxy, ok := boolEnv("CFL_USE_PROXY"); ok {
			if useProxy {
				proxy.UseRemotely = true
			} else {
				proxy = engine.ProxyConfig{}
			}
		}

		engine, err := docker.New(&engine.EngineConfig{
			Proxy: proxy,
		})
		if err != nil {
			return cli.ExitStatusUnknownError, err
		}
		defer engine.Close()

		// begin build specific code
		stager := forge.NewStager(engine)

		dropletPath := fmt.Sprintf("./%s.droplet", appName)
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

		droplet, err := stager.Stage(&forge.StageConfig{
			AppTar:        appTar,
			Cache:         cache,
			CacheEmpty:    cacheSize == 0,
			BuildpackZips: nil,
			Stack:         BuildStack,
			ForceDetect:   true,
			Color:         color.GreenString,
			AppConfig:     app,
		})
		if err != nil {
			return cli.ExitStatusUnknownError, err
		}
		defer droplet.Close()

		if err := streamOut(*sysFS, droplet, dropletPath); err != nil {
			return cli.ExitStatusUnknownError, err
		}

		return cli.ExitStatusSuccess, nil
	},
}

// TODO put in utils

func firstEnv(ks ...string) string {
	for _, k := range ks {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

func boolEnv(k string) (v, ok bool) {
	switch strings.TrimSpace(strings.ToLower(os.Getenv(k))) {
	case "true", "yes", "1":
		return true, true
	case "false", "no", "0":
		return false, true
	}
	return false, false
}


func streamOut(fs fs.FS, stream engine.Stream, path string) error {
	file, err := fs.WriteFile(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return stream.Out(file)
}
