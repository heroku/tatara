package cli

// Command holds a single subcommand
type Command struct {
	Name  string
	Flags []Flag

	Run func(context *Context) (exitStatus int, err error)
}
