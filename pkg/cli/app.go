package cli

// App is the top-level CLI application. It holds the Router
// and the CLIContext that provides dependencies to command handlers.
type App struct {
	router *Router
	ctx    *CLIContext
}

// NewApp creates a new CLI App with a Router.
// The CLIContext may be nil during initial setup — it is wired after
// config loading in the entry point.
func NewApp(ctx *CLIContext) *App {
	router := NewRouter(nil, nil, nil)
	return &App{
		router: router,
		ctx:    ctx,
	}
}

// Router returns the underlying Router, allowing callers to register
// commands or set pre-run hooks.
func (a *App) Router() *Router {
	return a.router
}

// Context returns the CLIContext.
func (a *App) Context() *CLIContext {
	return a.ctx
}

// SetContext sets the CLIContext. This is used when the context is built
// after config loading but before command execution.
func (a *App) SetContext(ctx *CLIContext) {
	a.ctx = ctx
}

// Run executes the router with the given arguments.
// Pass nil to use os.Args[1:].
func (a *App) Run(args []string) error {
	return a.router.Run(args)
}
