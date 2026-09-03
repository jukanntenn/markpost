// Package cliapp carries the CLI's shared runtime pieces: the dependency
// factory (gh-style lazy closures), and the error/exit-code vocabulary that
// maps action failures to printed messages.
package cliapp

import (
	"fmt"
	"runtime"
	"time"

	"markpost/cli/internal/agentenv"
	"markpost/cli/internal/api"
	"markpost/cli/internal/config"
	"markpost/cli/internal/iostreams"
)

// Factory is the dependency bag handed to every command, gh-style: each
// dependency is a lazy, memoized closure so `markpost version` never touches
// the network or the config file, and tests swap one closure without
// rebuilding the world.
type Factory struct {
	AppVersion string
	IO         *iostreams.IOStreams
	Now        func() time.Time

	// serverOverride carries the --server flag (whose EnvVars also folds in
	// MARKPOST_SERVER); it outranks the stored session's server.
	serverOverride string
	// configPath overrides DefaultPath(); tests point it at a temp file.
	configPath string

	Config     func() (*config.Config, error)
	SaveConfig func(*config.Config) error
	Client     func() (*api.Client, error)

	cfg    *config.Config
	client *api.Client
}

func NewFactory(version string, io *iostreams.IOStreams) *Factory {
	f := &Factory{
		AppVersion: version,
		IO:         io,
		Now:        time.Now,
	}
	f.Config = func() (*config.Config, error) {
		if f.cfg != nil {
			return f.cfg, nil
		}
		path := f.configPath
		if path == "" {
			var err error
			path, err = config.DefaultPath()
			if err != nil {
				return nil, err
			}
		}
		cfg, err := config.Load(path)
		if err != nil {
			return nil, err
		}
		f.cfg = cfg
		return cfg, nil
	}
	f.SaveConfig = func(cfg *config.Config) error {
		path := f.configPath
		if path == "" {
			var err error
			path, err = config.DefaultPath()
			if err != nil {
				return err
			}
		}
		if err := config.Save(path, cfg); err != nil {
			return err
		}
		f.cfg = cfg
		return nil
	}
	f.Client = func() (*api.Client, error) {
		if f.client != nil {
			return f.client, nil
		}
		cfg, err := f.Config()
		if err != nil {
			return nil, err
		}
		server := f.serverOverride
		if server == "" {
			if server, err = config.ResolveServer(cfg); err != nil {
				return nil, err
			}
		} else if server, err = config.NormalizeServer(server); err != nil {
			return nil, err
		}
		token, fromEnv := config.ResolveToken(cfg, server)

		client := api.New(server, nil)
		client.SetUserAgent(userAgent(f.AppVersion, agentenv.Detect()))
		if token != "" {
			refresh := ""
			if !fromEnv {
				refresh = cfg.Auth.RefreshToken
			}
			client.SetSession(token, refresh)
			if refresh != "" {
				client.SetTokensChanged(f.persistRefreshedTokens(cfg))
			}
		}
		f.client = client
		return client, nil
	}
	return f
}

// persistRefreshedTokens returns the TokensChanged callback that stores a
// refreshed pair. A refresh-failure loop must not lose the session over a
// write error, so the error is reported to stderr and swallowed.
func (f *Factory) persistRefreshedTokens(cfg *config.Config) func(access, refresh string, expiresAt time.Time) {
	return func(access, refresh string, expiresAt time.Time) {
		cfg.Auth.Token = access
		cfg.Auth.RefreshToken = refresh
		cfg.Auth.ExpiresAt = expiresAt.Unix()
		if err := f.SaveConfig(cfg); err != nil {
			fmt.Fprintf(f.IO.ErrOut, "markpost: saving refreshed session: %v\n", err)
		}
	}
}

// SetServerOverride records the --server flag before commands run.
func (f *Factory) SetServerOverride(server string) { f.serverOverride = server }

// SetConfigPath pins the config file location (tests).
func (f *Factory) SetConfigPath(path string) { f.configPath = path }

// Reset drops memoized config and client so later calls re-read state from
// disk (a command that just logged in wants the next Client() to see it).
func (f *Factory) Reset() {
	f.cfg = nil
	f.client = nil
}

// AuthenticatedClient returns a client that must carry a session, mapping
// "no credentials" to an AuthError so main exits 4 with a login hint instead
// of a bare 401.
func (f *Factory) AuthenticatedClient() (*api.Client, error) {
	client, err := f.Client()
	if err != nil {
		return nil, err
	}
	if client.AccessToken() == "" {
		return nil, &api.AuthError{Message: "not logged in"}
	}
	return client, nil
}

func userAgent(version, agent string) string {
	base := fmt.Sprintf("markpost-cli/%s (%s/%s)", version, runtime.GOOS, runtime.GOARCH)
	return base + agentenv.UserAgentSuffix(agent)
}

// UserAgent is the CLI's User-Agent string, detecting the driving agent so
// server-side telemetry can tell agent traffic from human traffic. Commands
// that build one-off clients (auth login) use this instead of the factory
// memo.
func UserAgent(version string) string {
	return userAgent(version, agentenv.Detect())
}
