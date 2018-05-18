package main

import (
	"fmt"

	"github.com/heroku/heroku-local/cli"
)

var cmdRun = cli.Command{
	Name: "run",

	Run: func(c *cli.Context) (int, error) {
    	fmt.Fprint(c.App.UserErr, "required arguments: <app directory>")

		return cli.ExitStatusSuccess, nil
	},
}
