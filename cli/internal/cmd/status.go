package cmd

import (
	"fmt"

	"markpost/cli/internal/cliapp"

	"github.com/urfave/cli/v2"
)

// StatusCommand is the smoke test: server identity plus health, readiness,
// and version in one shot.
func StatusCommand(f *cliapp.Factory) *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "Show server health, readiness, and version",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "json", Usage: "output the probe results as JSON"},
		},
		Action: func(c *cli.Context) error {
			client, err := f.Client()
			if err != nil {
				return err
			}
			health, err := client.Health(c.Context)
			if err != nil {
				return fmt.Errorf("health check failed: %w", err)
			}
			ready, err := client.Ready(c.Context)
			if err != nil {
				return fmt.Errorf("readiness check failed: %w", err)
			}
			version, err := client.Version(c.Context)
			if err != nil {
				return fmt.Errorf("version check failed: %w", err)
			}
			if c.Bool("json") {
				return printJSON(f.IO.Out, struct {
					Server  string `json:"server"`
					Health  string `json:"health"`
					Ready   string `json:"ready"`
					Version string `json:"version"`
				}{client.BaseURL(), health, ready, version}, f.IO.IsStdoutTTY())
			}
			fmt.Fprintf(f.IO.Out, "Server:  %s\n", client.BaseURL())
			fmt.Fprintf(f.IO.Out, "Health:  %s\n", health)
			fmt.Fprintf(f.IO.Out, "Ready:   %s\n", ready)
			fmt.Fprintf(f.IO.Out, "Version: %s\n", version)
			return nil
		},
	}
}

// VersionCommand prints the CLI's own version.
func VersionCommand(f *cliapp.Factory) *cli.Command {
	return &cli.Command{
		Name:   "version",
		Usage:  "Print the CLI version",
		Action: func(c *cli.Context) error { return versionRun(f) },
	}
}

func versionRun(f *cliapp.Factory) error {
	fmt.Fprintf(f.IO.Out, "markpost-cli version %s\n", f.AppVersion)
	return nil
}
