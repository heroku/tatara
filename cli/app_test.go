package cli

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecuteApp(t *testing.T) {
	testCommand := Command{
		Name: "testing",

		Run: func(c *Context) (int, error) {
			return 0, nil
		},
	}

	app := App{
		Commands: []Command{
			testCommand,
		},
	}

	exitStatus, err := app.Run([]string{"test_app", "testing"})
	assert.Equal(t, 0, exitStatus)
	assert.Nil(t, err)
}

func TestExecuteAppReturnOtherStatusCode(t *testing.T) {
	testCommand := Command{
		Name: "testing",

		Run: func(context *Context) (int, error) {
			return 79, nil
		},
	}

	app := App{
		Commands: []Command{
			testCommand,
		},
	}

	exitStatus, err := app.Run([]string{"test_app", "testing"})
	assert.Equal(t, 79, exitStatus)
	assert.Nil(t, err)
}

func TestExecuteAppReturnError(t *testing.T) {
	testCommand := Command{
		Name: "testing",

		Run: func(context *Context) (int, error) {
			return 1, errors.New("Invalid command")
		},
	}

	app := App{
		Commands: []Command{
			testCommand,
		},
	}

	exitStatus, err := app.Run([]string{"test_app", "testing"})
	assert.Equal(t, 1, exitStatus)
	assert.Equal(t, errors.New("Invalid command"), err)
}

func TestExecuteAppPassesArgs(t *testing.T) {
	testCommand := Command{
		Name: "testing",

		Run: func(context *Context) (int, error) {
			return 0, fmt.Errorf("%q", context.Args)
		},
	}

	app := App{
		Commands: []Command{
			testCommand,
		},
	}

	exitStatus, err := app.Run([]string{"test_app", "testing", "foobar", "hello.world"})
	assert.Equal(t, 0, exitStatus)
	assert.Equal(t, errors.New("[\"foobar\" \"hello.world\"]"), err)
}

func TestExecuteAppMissingCommand(t *testing.T) {
	testCommand := Command{
		Name: "testing",

		Run: func(context *Context) (int, error) {
			return 0, nil
		},
	}

	app := App{
		Commands: []Command{
			testCommand,
		},
	}

	exitStatus, err := app.Run([]string{"test_app", "missing_command"})
	assert.Equal(t, 1, exitStatus)
	assert.Equal(t, errors.New("Unknown command `missing_command`"), err)
}

func TestExecuteAppNoCommand(t *testing.T) {
	testCommand := Command{
		Name: "testing",

		Run: func(context *Context) (int, error) {
			return 0, nil
		},
	}

	app := App{
		Commands: []Command{
			testCommand,
		},
	}

	exitStatus, err := app.Run([]string{"test_app"})
	assert.Equal(t, 2, exitStatus)
	assert.Equal(t, errors.New("Please specify a command"), err)
}

func TestExecuteAppHasFlags(t *testing.T) {
	testCommand := Command{
		Name: "testing",
		Run: func(context *Context) (int, error) {
			flagValue := context.Flags.String("hello")

			if flagValue != "world" {
				return 1, fmt.Errorf("No flag --hello found. Got `%s`", flagValue)
			}

			return 0, nil
		},
	}

	app := App{
		Commands: []Command{
			testCommand,
		},
		Flags: []Flag{
			StringFlag{
				Name:  "hello",
				Value: "",
				Usage: "Hello world",
			},
		},
	}

	exitStatus, err := app.Run([]string{"test_app", "testing", "--hello=world"})
	assert.Equal(t, 0, exitStatus)
	assert.Nil(t, err)
}

func TestExecuteAppUnknownFlags(t *testing.T) {
	testCommand := Command{
		Name: "testing",
		Run: func(context *Context) (int, error) {
			return 0, nil
		},
	}

	app := App{
		Commands: []Command{
			testCommand,
		},
	}

	exitStatus, err := app.Run([]string{"test_app", "testing", "--hello=world"})
	assert.Equal(t, 1, exitStatus)
	assert.Equal(t, errors.New("unknown flag: --hello"), err)
}

func TestExecuteAppCommandHasFlags(t *testing.T) {
	testCommand := Command{
		Name: "testing",
		Run: func(context *Context) (int, error) {
			flagValue := context.Flags.String("hello")

			if flagValue != "world" {
				return 1, fmt.Errorf("No flag --hello found. Got `%s`", flagValue)
			}

			return 0, nil
		},
		Flags: []Flag{
			StringFlag{
				Name:  "hello",
				Value: "",
				Usage: "Hello world",
			},
		},
	}

	app := App{
		Commands: []Command{
			testCommand,
		},
	}

	exitStatus, err := app.Run([]string{"test_app", "testing", "--hello=world"})
	assert.Equal(t, 0, exitStatus)
	assert.Nil(t, err)
}
