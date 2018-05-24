package cli

// Context holds the context of a command execution
type Context struct {
	App         *App
	CommandName string
	Command     *Command
	Flags       *FlagSet
	Args        []string
	Exit 		<-chan struct{}
}
