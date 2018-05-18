package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlagSetString(t *testing.T) {
	flags := []Flag{
		StringFlag{
			Name:  "hello",
			Value: "",
			Usage: "hello world",
		},
	}
	flagSet, err := NewFlagSet("test", []string{"test", "--hello=world"}, flags)

	assert.Nil(t, err)
	assert.Equal(t, "world", flagSet.String("hello"))
}

func TestFlagSetInt(t *testing.T) {
	flags := []Flag{
		IntFlag{
			Name:  "hello",
			Value: 1,
			Usage: "hello world",
		},
	}
	flagSet, err := NewFlagSet("test", []string{"test", "--hello=15"}, flags)

	assert.Nil(t, err)
	assert.Equal(t, 15, flagSet.Int("hello"))
}

func TestFlagSetBool(t *testing.T) {
	flags := []Flag{
		BoolFlag{
			Name:  "hello",
			Usage: "hello world",
		},
	}
	flagSet, err := NewFlagSet("test", []string{"test", "--hello"}, flags)

	assert.Nil(t, err)
	assert.Equal(t, true, flagSet.Bool("hello"))
}

func TestFlagSetStringSlice(t *testing.T) {
	flags := []Flag{
		StringSliceFlag{
			Name:  "hello",
			Usage: "hello world",
		},
	}
	flagSet, err := NewFlagSet("test", []string{"test", "--hello=foo", "--hello=bar"}, flags)

	assert.Nil(t, err)
	assert.Equal(t, []string{"foo", "bar"}, flagSet.StringSlice("hello"))
}
