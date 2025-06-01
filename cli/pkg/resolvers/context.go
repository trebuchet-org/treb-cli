package resolvers

// Context holds the resolver configuration and state
type Context struct {
	interactive bool
	projectRoot string
}

// NewContext creates a new resolver context
func NewContext(projectRoot string, interactive bool) *Context {
	return &Context{
		interactive: interactive,
		projectRoot: projectRoot,
	}
}

// IsInteractive returns whether the context is in interactive mode
func (c *Context) IsInteractive() bool {
	return c.interactive
}

// ProjectRoot returns the project root directory
func (c *Context) ProjectRoot() string {
	return c.projectRoot
}
