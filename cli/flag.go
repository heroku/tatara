package cli

import (
	"fmt"
	"strconv"

	flag "github.com/ogier/pflag"
)

// FlagSet encapsulates the args flags
type FlagSet struct {
	*flag.FlagSet
}

// Flag is the basic interface for all flags
type Flag interface {
	Apply(*flag.FlagSet)
}

// NewFlagSet creates a new flag.FlagSet object and parses the flags
func NewFlagSet(name string, args []string, flags []Flag) (*FlagSet, error) {
	set := flag.NewFlagSet(name, flag.ContinueOnError)
	flagSet := &FlagSet{set}

	for _, flag := range flags {
		flag.Apply(set)
	}

	err := set.Parse(args)
	if err != nil {
		return nil, err
	}

	return flagSet, nil
}

// String looks up a flag as a string
func (f *FlagSet) String(name string) string {
	s := f.Lookup(name)
	if s != nil {
		return s.Value.String()
	}
	return ""
}

// Int looks up a flag as integer
func (f *FlagSet) Int(name string) int {
	value := f.String(name)
	if value != "" {
		val, err := strconv.Atoi(value)
		if err != nil {
			return 0
		}
		return val
	}
	return 0
}

// Bool looks up a flag as boolean
func (f *FlagSet) Bool(name string) bool {
	value := f.String(name)
	if value != "" {
		val, err := strconv.ParseBool(value)
		if err != nil {
			return false
		}
		return val
	}
	return false
}

// an opaque type for []string to satisfy flag.Value
type stringSlice []string

func (slice *stringSlice) Set(value string) error {
	*slice = append(*slice, value)
	return nil
}
func (slice *stringSlice) String() string {
	return fmt.Sprintf("%s", *slice)
}
func (slice *stringSlice) Value() []string {
	return *slice
}

// StringSlice looks up a flag as an array of strings
func (f *FlagSet) StringSlice(name string) []string {
	s := f.Lookup(name)
	if s != nil {
		return (s.Value.(*stringSlice)).Value()
	}
	return nil
}

/*
 * Below are all specific flag types
 */

// StringFlag holds a single args string flag
type StringFlag struct {
	Name  string
	Value string
	Usage string
}

// Apply applies a flag to the FlagSet
func (flag StringFlag) Apply(set *flag.FlagSet) {
	set.String(flag.Name, flag.Value, flag.Usage)
}

// IntFlag holds a single args integer flag
type IntFlag struct {
	Name  string
	Value int
	Usage string
}

// Apply applies a flag to the FlagSet
func (flag IntFlag) Apply(set *flag.FlagSet) {
	set.Int(flag.Name, flag.Value, flag.Usage)
}

// BoolFlag holds a single args boolean flag
type BoolFlag struct {
	Name  string
	Usage string
}

// Apply applies a boolean flag to the FlagSet
func (flag BoolFlag) Apply(set *flag.FlagSet) {
	set.Bool(flag.Name, false, flag.Usage)
}

// StringSliceFlag holds a single args string flag
type StringSliceFlag struct {
	Name  string
	Value *stringSlice
	Usage string
}

// Apply applies a flag to the FlagSet
func (flag StringSliceFlag) Apply(set *flag.FlagSet) {
	if flag.Value == nil {
		flag.Value = &stringSlice{}
	}

	set.Var(flag.Value, flag.Name, flag.Usage)
}
