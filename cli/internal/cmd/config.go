package cmd

import (
	"fmt"

	"markpost/cli/internal/cliapp"
	"markpost/cli/internal/config"

	"github.com/urfave/cli/v2"
)

// ConfigCommand reads and writes local CLI preferences.
func ConfigCommand(f *cliapp.Factory) *cli.Command {
	return &cli.Command{
		Name:        "config",
		Usage:       "Read or write local CLI configuration",
		Subcommands: []*cli.Command{configGetCommand(f), configSetCommand(f)},
		Action:      groupAction("config"),
	}
}

// configKeys is the allowlist of settable keys; unknown keys are usage
// errors rather than silently persisted typos.
var configKeys = map[string]struct{}{
	"server": {},
}

func configGetCommand(f *cliapp.Factory) *cli.Command {
	return &cli.Command{
		Name:      "get",
		Usage:     "Print a configuration value",
		UsageText: "markpost config get <key>",
		ArgsUsage: "<key>",
		Action: func(c *cli.Context) error {
			if c.NArg() != 1 {
				return cliapp.FlagErrorf("config get requires exactly one <key> argument")
			}
			key := c.Args().First()
			if _, ok := configKeys[key]; !ok {
				return cliapp.FlagErrorf("unknown config key %q (available: server)", key)
			}
			cfg, err := f.Config()
			if err != nil {
				return err
			}
			fmt.Fprintln(f.IO.Out, cfg.Auth.Server)
			return nil
		},
	}
}

func configSetCommand(f *cliapp.Factory) *cli.Command {
	return &cli.Command{
		Name:      "set",
		Usage:     "Write a configuration value",
		UsageText: "markpost config set <key> <value>",
		ArgsUsage: "<key> <value>",
		Action: func(c *cli.Context) error {
			if c.NArg() != 2 {
				return cliapp.FlagErrorf("config set requires <key> and <value> arguments")
			}
			key, value := c.Args().Get(0), c.Args().Get(1)
			if _, ok := configKeys[key]; !ok {
				return cliapp.FlagErrorf("unknown config key %q (available: server)", key)
			}
			normalized, err := config.NormalizeServer(value)
			if err != nil {
				return cliapp.FlagErrorf("%s", err)
			}
			cfg, err := f.Config()
			if err != nil {
				return err
			}
			cfg.Auth.Server = normalized
			if err := f.SaveConfig(cfg); err != nil {
				return err
			}
			fmt.Fprintf(f.IO.Out, "%s = %s\n", key, normalized)
			return nil
		},
	}
}
