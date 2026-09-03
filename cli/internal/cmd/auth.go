package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"markpost/cli/internal/api"
	"markpost/cli/internal/cliapp"
	"markpost/cli/internal/config"
	"markpost/cli/internal/iostreams"

	"github.com/urfave/cli/v2"
	"golang.org/x/term"
)

// AuthCommand groups session management.
func AuthCommand(f *cliapp.Factory) *cli.Command {
	return &cli.Command{
		Name:        "auth",
		Usage:       "Manage the login session with a markpost server",
		Subcommands: []*cli.Command{authLoginCommand(f), authStatusCommand(f), authTokenCommand(f), authLogoutCommand(f)},
		Action:      groupAction("auth"),
	}
}

func authLoginCommand(f *cliapp.Factory) *cli.Command {
	return &cli.Command{
		Name:      "login",
		Usage:     "Log in and store the session",
		UsageText: "markpost auth login [--server <url>] [--username <name>] [--password <pw> | --password-stdin | --token <token>]",
		Description: `Authenticates against the markpost server and stores the session
(including the refresh token) in the config file for automatic renewal.

Non-interactive callers must pass --username with --password (or
--password-stdin), or --token for an existing access token. With a terminal,
missing values are prompted for. A --token session has no refresh token: it
stops working when the access token expires.`,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "server", Usage: "server base URL; saved with the session"},
			&cli.StringFlag{Name: "username", Usage: "account username"},
			&cli.StringFlag{Name: "password", Usage: "account password (prefer --password-stdin in scripts)"},
			&cli.BoolFlag{Name: "password-stdin", Usage: "read the password from stdin"},
			&cli.StringFlag{Name: "token", Usage: "use an existing access token instead of credentials"},
		},
		Action: func(c *cli.Context) error {
			return authLoginRun(f, c, authLoginOptions{
				Server:        c.String("server"),
				Username:      c.String("username"),
				Password:      c.String("password"),
				PasswordStdin: c.Bool("password-stdin"),
				Token:         c.String("token"),
			})
		},
	}
}

type authLoginOptions struct {
	Server        string
	Username      string
	Password      string
	PasswordStdin bool
	Token         string
}

func authLoginRun(f *cliapp.Factory, c *cli.Context, opts authLoginOptions) error {
	io := f.IO
	ctx := c.Context

	if opts.Token != "" && (opts.Username != "" || opts.Password != "" || opts.PasswordStdin) {
		return cliapp.FlagErrorf("--token cannot be combined with --username/--password/--password-stdin")
	}

	cfg, err := f.Config()
	if err != nil {
		return err
	}
	server := opts.Server
	if server == "" {
		if server, err = config.ResolveServer(cfg); err != nil {
			return cliapp.FlagErrorf("%s", err)
		}
	}
	server, err = config.NormalizeServer(server)
	if err != nil {
		return cliapp.FlagErrorf("%s", err)
	}

	client := api.New(server, nil)
	client.SetUserAgent(cliapp.UserAgent(f.AppVersion))

	if opts.Token != "" {
		// Verify the token before storing it so `auth login --token` cannot
		// succeed with garbage. Retention is the cheapest authed endpoint.
		client.SetSession(opts.Token, "")
		if _, err := client.Retention(ctx); err != nil {
			return fmt.Errorf("token rejected by %s: %w", server, err)
		}
		cfg.Auth = config.Auth{
			Server:    server,
			Token:     opts.Token,
			ExpiresAt: 0,
		}
		if err := f.SaveConfig(cfg); err != nil {
			return err
		}
		fmt.Fprintf(io.Out, "Logged in to %s with token\n", server)
		return nil
	}

	username := opts.Username
	if username == "" {
		if !io.CanPrompt() {
			return cliapp.FlagErrorf("--username is required when not interactive")
		}
		if username, err = promptLine(io, "Username: "); err != nil {
			return err
		}
	}
	password := opts.Password
	switch {
	case opts.PasswordStdin:
		if io.IsStdinTTY() {
			return cliapp.FlagErrorf("--password-stdin requires piped stdin")
		}
		raw, err := io.ReadAll(io.In)
		if err != nil {
			return err
		}
		password = strings.TrimRight(raw, "\r\n")
	case password == "" && io.CanPrompt():
		if password, err = promptPassword("Password: "); err != nil {
			return err
		}
	case password == "":
		return cliapp.FlagErrorf("--password or --password-stdin is required when not interactive")
	}
	if username == "" || password == "" {
		return cliapp.FlagErrorf("username and password must not be empty")
	}

	login, err := client.Login(ctx, username, password)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	cfg.Auth = config.Auth{
		Server:       server,
		UserID:       login.User.ID,
		Username:     login.User.Username,
		Name:         login.User.Name,
		Role:         login.User.Role,
		Token:        login.Token,
		RefreshToken: login.RefreshToken,
		ExpiresAt:    f.Now().Add(time.Duration(login.ExpiresIn) * time.Second).Unix(),
	}
	if err := f.SaveConfig(cfg); err != nil {
		return err
	}
	fmt.Fprintf(io.Out, "Logged in as %s to %s\n", login.User.Username, server)
	return nil
}

func authStatusCommand(f *cliapp.Factory) *cli.Command {
	return &cli.Command{
		Name:   "status",
		Usage:  "Show the session and verify it against the server",
		Action: func(c *cli.Context) error { return authStatusRun(f, c) },
	}
}

func authStatusRun(f *cliapp.Factory, c *cli.Context) error {
	io := f.IO
	cfg, err := f.Config()
	if err != nil {
		return err
	}
	server, err := config.ResolveServer(cfg)
	if err != nil {
		return err
	}
	token, fromEnv := config.ResolveToken(cfg, server)

	fmt.Fprintf(io.Out, "Server: %s\n", server)
	switch {
	case token == "":
		fmt.Fprintln(io.Out, "Session: none")
		return &api.AuthError{Message: "not logged in"}
	case fromEnv:
		fmt.Fprintln(io.Out, "Session: MARKPOST_TOKEN (no refresh)")
	default:
		fmt.Fprintf(io.Out, "Session: %s\n", cfg.Auth.Username)
		if cfg.Auth.ExpiresAt > 0 {
			expires := time.Unix(cfg.Auth.ExpiresAt, 0)
			fmt.Fprintf(io.Out, "Token expires: %s (%s)\n", expires.Format(time.RFC3339), humanUntil(f.Now(), expires))
		}
	}

	client, err := f.Client()
	if err != nil {
		return err
	}
	if _, err := client.Retention(c.Context); err != nil {
		fmt.Fprintf(io.Out, "Verification: FAILED\n")
		return fmt.Errorf("session verification failed: %w", err)
	}
	fmt.Fprintln(io.Out, "Verification: ok")
	return nil
}

func authTokenCommand(f *cliapp.Factory) *cli.Command {
	return &cli.Command{
		Name:  "token",
		Usage: "Print the current access token (for piping)",
		Action: func(c *cli.Context) error {
			cfg, err := f.Config()
			if err != nil {
				return err
			}
			server, err := config.ResolveServer(cfg)
			if err != nil {
				return err
			}
			token, _ := config.ResolveToken(cfg, server)
			if token == "" {
				return &api.AuthError{Message: "not logged in"}
			}
			fmt.Fprintln(f.IO.Out, token)
			return nil
		},
	}
}

func authLogoutCommand(f *cliapp.Factory) *cli.Command {
	return &cli.Command{
		Name:  "logout",
		Usage: "Revoke the session server-side and clear local credentials",
		Action: func(c *cli.Context) error {
			cfg, err := f.Config()
			if err != nil {
				return err
			}
			server, err := config.ResolveServer(cfg)
			if err != nil {
				return err
			}
			if token, _ := config.ResolveToken(cfg, server); token != "" {
				if client, err := f.Client(); err == nil {
					// Best effort: a token the server already rejected must
					// not block local cleanup.
					_ = client.Logout(c.Context)
				}
			}
			cfg.Auth = config.Auth{Server: cfg.Auth.Server}
			if err := f.SaveConfig(cfg); err != nil {
				return err
			}
			fmt.Fprintf(f.IO.Out, "Logged out of %s\n", server)
			return nil
		},
	}
}

// promptLine asks on stderr (keeps stdout parseable) and reads one line.
func promptLine(io *iostreams.IOStreams, label string) (string, error) {
	fmt.Fprint(io.ErrOut, label)
	return readLine(io.In)
}

// readLine reads one line (without the trailing newline), one byte at a time
// so nothing after the line is swallowed from a pipe.
func readLine(r io.Reader) (string, error) {
	var sb strings.Builder
	buf := make([]byte, 1)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			if buf[0] == '\n' {
				return strings.TrimSuffix(sb.String(), "\r"), nil
			}
			sb.WriteByte(buf[0])
		}
		if err == io.EOF {
			return sb.String(), nil
		}
		if err != nil {
			return sb.String(), err
		}
	}
}

// promptPassword reads a password without echo; it goes straight to the
// terminal fd because that is the only thing x/term can silence.
func promptPassword(label string) (string, error) {
	fmt.Fprint(os.Stderr, label)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// humanUntil renders coarse remaining time like "in 2h13m".
func humanUntil(now, until time.Time) string {
	d := until.Sub(now)
	if d < 0 {
		return "expired"
	}
	return "in " + d.Truncate(time.Minute).String()
}
