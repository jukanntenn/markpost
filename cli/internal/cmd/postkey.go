package cmd

import (
	"fmt"

	"markpost/cli/internal/cliapp"

	"github.com/urfave/cli/v2"
)

// PostKeyCommand manages the publishing key.
func PostKeyCommand(f *cliapp.Factory) *cli.Command {
	return &cli.Command{
		Name:        "post-key",
		Usage:       "Show or rotate the post key used to publish",
		Subcommands: []*cli.Command{postKeyShowCommand(f), postKeyRotateCommand(f)},
		Action:      groupAction("post-key"),
	}
}

func postKeyShowCommand(f *cliapp.Factory) *cli.Command {
	return &cli.Command{
		Name:  "show",
		Usage: "Print the current post key",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "json", Usage: "output {post_key, created_at} as JSON"},
		},
		Action: func(c *cli.Context) error {
			client, err := f.AuthenticatedClient()
			if err != nil {
				return err
			}
			key, err := client.PostKey(c.Context)
			if err != nil {
				return err
			}
			if c.Bool("json") {
				return printJSON(f.IO.Out, key, f.IO.IsStdoutTTY())
			}
			fmt.Fprintln(f.IO.Out, key.PostKey)
			return nil
		},
	}
}

func postKeyRotateCommand(f *cliapp.Factory) *cli.Command {
	return &cli.Command{
		Name:  "rotate",
		Usage: "Issue a new post key (the old one stops working immediately)",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "json", Usage: "output {post_key} as JSON"},
		},
		Action: func(c *cli.Context) error {
			client, err := f.AuthenticatedClient()
			if err != nil {
				return err
			}
			key, err := client.RotatePostKey(c.Context)
			if err != nil {
				return err
			}
			if c.Bool("json") {
				return printJSON(f.IO.Out, map[string]string{"post_key": key}, f.IO.IsStdoutTTY())
			}
			fmt.Fprintln(f.IO.Out, key)
			return nil
		},
	}
}
