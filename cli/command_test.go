package cli

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecuteCommand(t *testing.T) {
	command := Command{
		Run: func(c *Context) (int, error) {
			return 0, nil
		},
	}

	exitStatus, err := command.Run(&Context{})
	assert.Equal(t, 0, exitStatus)
	assert.Nil(t, err)
}

func TestExecuteCommandReturnOtherStatusCode(t *testing.T) {
	command := Command{
		Run: func(c *Context) (int, error) {
			return 79, nil
		},
	}

	exitStatus, err := command.Run(&Context{})
	assert.Equal(t, 79, exitStatus)
	assert.Nil(t, err)
}

func TestExecuteCommandReturnError(t *testing.T) {
	command := Command{
		Run: func(c *Context) (int, error) {
			return 1, errors.New("Invalid command")
		},
	}

	exitStatus, err := command.Run(&Context{})
	assert.Equal(t, 1, exitStatus)
	assert.Equal(t, errors.New("Invalid command"), err)
}
