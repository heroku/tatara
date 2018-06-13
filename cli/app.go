package cli

import (
	"errors"
	"fmt"
	"io"
)

// Constants for exit statuses
const (
	ExitStatusSuccess = iota
	ExitStatusUnknownError
	ExitStatusInvalidArgs
)

// App is the main structure for any CLI application
type App struct {
	Commands []Command
	Flags    []Flag

	// UserOut logs messages to end-users
	UserOut io.Writer
	// UserErr logs errors to end-users
	UserErr io.Writer
	// InternalOut logs messages to ops
	InternalOut io.Writer

	Before func(c *Context) error

	Exit <-chan struct{}
}

// Run executed the command with the provided arguments
func (a *App) Run(args []string) (int, error) {
	if len(args) < 2 {
		return ExitStatusInvalidArgs, errors.New("Please specify a command")
	}

	appName, commandName, args := args[0], args[1], args[2:]
	exitStatus := ExitStatusUnknownError

	command, err := a.findCommand(commandName)
	if err != nil {
		return exitStatus, err
	}

	flags := append(a.Flags, command.Flags...)
	flagSet, err := NewFlagSet(appName, args, flags)

	if err != nil {
		return exitStatus, err
	}

	context := &Context{
		App:         a,
		CommandName: commandName,
		Command:     command,
		Flags:       flagSet,
		Args:        flagSet.Args(),
		Exit:        a.Exit,
	}

	if a.Before != nil {
		err := a.Before(context)
		if err != nil {
			return exitStatus, err
		}
	}

	exitStatus, err = command.Run(context)

	return exitStatus, err
}

func (a *App) findCommand(commandName string) (*Command, error) {
	for _, cmd := range a.Commands {
		if cmd.Name == commandName {
			return &cmd, nil
		}
	}

	err := fmt.Errorf("Unknown command `%s`", commandName)
	return nil, err
}
