package cmd

import (
	"fmt"
	"os"

	"markpost/cli/internal/api"
	"markpost/cli/internal/cliapp"

	"github.com/urfave/cli/v2"
)

// APICommand is the generic passthrough (gh api style): any endpoint the
// typed commands do not cover is reachable through it, which keeps the CLI
// useful the day the server grows a new route.
func APICommand(f *cliapp.Factory) *cli.Command {
	return &cli.Command{
		Name:      "api",
		Usage:     "Make an authenticated HTTP request to the server API",
		UsageText: "markpost api <endpoint> [-X <method>] [--input <file|->]",
		ArgsUsage: "<endpoint>",
		Description: `endpoint is a path; relative paths resolve against /api/v1 (e.g.
"me/retention" hits /api/v1/me/retention). Absolute paths and full URLs are
used verbatim, so public routes work too: markpost api /health.

The response body is printed verbatim to stdout. Use --input to send a body
(a file path, or - for stdin); it is sent as application/json.`,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "method", Aliases: []string{"X"}, Value: "GET", Usage: "HTTP method"},
			&cli.StringFlag{Name: "input", Aliases: []string{"i"}, Usage: "request body file, or - for stdin"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() != 1 {
				return cliapp.FlagErrorf("api requires exactly one <endpoint> argument")
			}
			method := c.String("method")
			if !validMethod(method) {
				return cliapp.FlagErrorf("invalid --method %q", method)
			}
			client, err := f.Client()
			if err != nil {
				return err
			}
			var body []byte
			if input := c.String("input"); input != "" {
				if input == "-" {
					raw, err := f.IO.ReadAllStdin()
					if err != nil {
						return err
					}
					body = []byte(raw)
				} else if body, err = os.ReadFile(input); err != nil {
					return fmt.Errorf("read %s: %w", input, err)
				}
			}
			status, respBody, _, err := client.Passthrough(c.Context, api.PassthroughRequest{
				Method: method,
				Path:   api.ResolveAPIPath(c.Args().First()),
				Body:   body,
				Authed: client.AccessToken() != "",
			})
			if err != nil {
				return err
			}
			if len(respBody) > 0 {
				fmt.Fprint(f.IO.Out, string(respBody))
			}
			if status >= 400 {
				return fmt.Errorf("HTTP %d from %s", status, c.Args().First())
			}
			return nil
		},
	}
}

func validMethod(m string) bool {
	switch m {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
		return true
	default:
		return false
	}
}
