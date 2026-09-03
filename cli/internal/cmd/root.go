// Package cmd assembles the markpost CLI: the urfave/cli app, the command
// tree, and Main — the single place that turns action errors into printed
// messages and exit codes (mirrors gh's internal/ghcmd).
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"markpost/cli/internal/agentenv"
	"markpost/cli/internal/api"
	"markpost/cli/internal/cliapp"
	"markpost/cli/internal/config"
	"markpost/cli/internal/iostreams"

	"github.com/urfave/cli/v2"
)

// NewApp builds the command tree around f.
func NewApp(f *cliapp.Factory) *cli.App {
	return &cli.App{
		Name:                 "markpost",
		Usage:                "Publish and manage markdown posts on a markpost server",
		UsageText:            "markpost [global options] command [command options] [arguments...]",
		Version:              f.AppVersion,
		Suggest:              true,
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "server",
				Usage:   "markpost server base URL, e.g. https://mp.example.com",
				EnvVars: []string{"MARKPOST_SERVER"},
			},
		},
		Commands: []*cli.Command{
			AuthCommand(f),
			PostsCommand(f),
			PostKeyCommand(f),
			APICommand(f),
			StatusCommand(f),
			ConfigCommand(f),
			VersionCommand(f),
		},
		Action: groupAction("markpost"),
	}
}

// applyUsageErrorSink walks the command tree setting OnUsageError on every
// command: urfave only copies App.OnUsageError onto the root command, so
// without this a subcommand's flag error would print "Incorrect Usage" plus
// help to stdout and bypass Main's (agent-aware) classification.
func applyUsageErrorSink(cmds []*cli.Command, fn cli.OnUsageErrorFunc) {
	for _, c := range cmds {
		c.OnUsageError = fn
		applyUsageErrorSink(c.Subcommands, fn)
	}
}

// Main runs the CLI end to end and returns the process exit code. All error
// printing happens here so commands simply return errors. Classification
// order matters: usage errors first (they carry their own help), then auth
// (exit 4 with a login hint), then cancellation, then everything else.
func Main(version string, args []string, io *iostreams.IOStreams, configPath string) int {
	f := cliapp.NewFactory(version, io)
	if configPath != "" {
		f.SetConfigPath(configPath)
	}
	app := NewApp(f)
	app.Writer = io.Out
	app.ErrWriter = io.ErrOut
	onUsageError := func(_ *cli.Context, err error, _ bool) error {
		return &cliapp.FlagError{Err: err}
	}
	app.OnUsageError = onUsageError
	applyUsageErrorSink(app.Commands, onUsageError)
	// Errors must flow back to RunContext's return value: urfave's default
	// handler would print and os.Exit itself, racing Main's classification.
	// The context is captured so the agent help fallback below can render it.
	var runCtx *cli.Context
	app.ExitErrHandler = func(c *cli.Context, _ error) { runCtx = c }

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	err := app.RunContext(ctx, args)
	if err == nil {
		return cliapp.ExitOK
	}

	var flagErr *cliapp.FlagError
	if errors.As(err, &flagErr) {
		fmt.Fprintf(io.ErrOut, "markpost: %s\n", flagErr.Error())
		if agentenv.Detect() != "" && runCtx != nil {
			// Agents self-correct from the full command list and examples
			// faster than from a one-line usage error.
			app.Writer = io.ErrOut
			_ = cli.ShowAppHelp(runCtx)
			app.Writer = io.Out
		}
		return cliapp.ExitError
	}
	if api.IsAuthError(err) || errors.Is(err, config.ErrNoServer) {
		fmt.Fprintf(io.ErrOut, "markpost: %s\n", err)
		fmt.Fprintln(io.ErrOut, "Try 'markpost auth login' (see 'markpost auth login --help'), or set MARKPOST_SERVER and MARKPOST_TOKEN.")
		return cliapp.ExitAuth
	}
	if errors.Is(err, context.Canceled) {
		fmt.Fprintln(io.ErrOut, "markpost: canceled")
		return cliapp.ExitCancel
	}
	fmt.Fprintf(io.ErrOut, "markpost: %s\n", err)
	return cliapp.ExitError
}

// groupAction backs both the root command and command groups: bare
// invocation shows help; a stray first argument means the user (or agent)
// mistyped a subcommand, reported as a usage error.
func groupAction(name string) func(*cli.Context) error {
	return func(c *cli.Context) error {
		if c.NArg() > 0 {
			return cliapp.FlagErrorf("unknown command %q for %q", c.Args().First(), name)
		}
		if name == "markpost" {
			return cli.ShowAppHelp(c)
		}
		return cli.ShowCommandHelp(c, name)
	}
}
