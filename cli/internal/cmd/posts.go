package cmd

import (
	"fmt"
	"os"
	"strings"

	"markpost/cli/internal/api"
	"markpost/cli/internal/cliapp"
	"markpost/cli/internal/iostreams"

	"github.com/urfave/cli/v2"
)

// PostsCommand groups post publishing and management.
func PostsCommand(f *cliapp.Factory) *cli.Command {
	return &cli.Command{
		Name:        "posts",
		Aliases:     []string{"post"},
		Usage:       "Publish and manage markdown posts",
		Subcommands: []*cli.Command{postsCreateCommand(f), postsListCommand(f), postsViewCommand(f), postsDeleteCommand(f)},
		Action:      groupAction("posts"),
	}
}

func postsCreateCommand(f *cliapp.Factory) *cli.Command {
	return &cli.Command{
		Name:      "create",
		Usage:     "Publish a markdown post and print its URL",
		UsageText: "markpost posts create -t <title> (-f <file|-> | <body...>) [--post-key <key>] [--json]",
		Description: `Creates a post on the configured server. The body comes from --file
(a path, or "-" for stdin), or the positional arguments joined by newlines, or
piped stdin, in that order.

Publishing uses the post key: --post-key / MARKPOST_POST_KEY when provided,
otherwise the key is fetched with the logged-in session.`,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "title", Aliases: []string{"t"}, Usage: "post title", Required: true},
			&cli.StringFlag{Name: "file", Aliases: []string{"f"}, Usage: "markdown body file, or - for stdin"},
			&cli.StringFlag{Name: "post-key", Usage: "publish under this post key", EnvVars: []string{"MARKPOST_POST_KEY"}},
			&cli.BoolFlag{Name: "json", Usage: "output {qid, url} as JSON"},
		},
		Action: func(c *cli.Context) error {
			return postsCreateRun(f, c, postsCreateOptions{
				Title:   c.String("title"),
				File:    c.String("file"),
				PostKey: c.String("post-key"),
				JSON:    c.Bool("json"),
				Args:    c.Args().Slice(),
			})
		},
	}
}

type postsCreateOptions struct {
	Title   string
	File    string
	PostKey string
	JSON    bool
	Args    []string
}

func postsCreateRun(f *cliapp.Factory, c *cli.Context, opts postsCreateOptions) error {
	io := f.IO
	body, err := resolveBody(io, opts.File, opts.Args)
	if err != nil {
		return err
	}

	client, err := f.Client()
	if err != nil {
		return err
	}
	postKey := opts.PostKey
	if postKey == "" {
		authed, err := f.AuthenticatedClient()
		if err != nil {
			return err
		}
		key, err := authed.PostKey(c.Context)
		if err != nil {
			return fmt.Errorf("fetching post key: %w", err)
		}
		postKey = key.PostKey
	}
	qid, err := client.CreatePost(c.Context, postKey, opts.Title, body)
	if err != nil {
		return err
	}
	url := postURL(client.BaseURL(), qid)
	if opts.JSON {
		return printJSON(io.Out, map[string]string{"qid": qid, "url": url}, io.IsStdoutTTY())
	}
	fmt.Fprintln(io.Out, url)
	return nil
}

// resolveBody picks the post body: --file (or -) over positional arguments
// over piped stdin. An empty body is rejected up front — the server would
// reject it anyway, but with a round trip.
func resolveBody(io *iostreams.IOStreams, file string, args []string) (string, error) {
	var body string
	switch {
	case file != "":
		if file == "-" {
			if io.IsStdinTTY() {
				return "", cliapp.FlagErrorf("--file - requires piped stdin")
			}
			raw, err := io.ReadAllStdin()
			if err != nil {
				return "", err
			}
			body = raw
		} else {
			data, err := os.ReadFile(file)
			if err != nil {
				return "", fmt.Errorf("read %s: %w", file, err)
			}
			body = string(data)
		}
	case len(args) > 0:
		body = strings.Join(args, "\n")
	default:
		if io.IsStdinTTY() {
			return "", cliapp.FlagErrorf("provide the body via --file, a positional argument, or piped stdin")
		}
		raw, err := io.ReadAllStdin()
		if err != nil {
			return "", err
		}
		body = raw
	}
	if strings.TrimSpace(body) == "" {
		return "", cliapp.FlagErrorf("body is empty")
	}
	return body, nil
}

func postURL(baseURL, qid string) string {
	return strings.TrimSuffix(baseURL, "/") + "/" + qid
}

func postsListCommand(f *cliapp.Factory) *cli.Command {
	return &cli.Command{
		Name:      "list",
		Aliases:   []string{"ls"},
		Usage:     "List your posts",
		UsageText: "markpost posts list [--search <text>] [--page <n>] [--limit <n>] [--json]",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "search", Usage: "filter by title or body text"},
			&cli.IntFlag{Name: "page", Value: 1, Usage: "page number (min 1)"},
			&cli.IntFlag{Name: "limit", Value: 20, Usage: "items per page"},
			&cli.BoolFlag{Name: "json", Usage: "output the raw list envelope as JSON"},
		},
		Action: func(c *cli.Context) error {
			if c.Int("page") < 1 {
				return cliapp.FlagErrorf("--page must be >= 1")
			}
			if c.Int("limit") < 1 {
				return cliapp.FlagErrorf("--limit must be >= 1")
			}
			client, err := f.AuthenticatedClient()
			if err != nil {
				return err
			}
			list, err := client.ListPosts(c.Context, api.ListPostsParams{
				Search: c.String("search"),
				Page:   c.Int("page"),
				Limit:  c.Int("limit"),
			})
			if err != nil {
				return err
			}
			if c.Bool("json") {
				return printJSON(f.IO.Out, list, f.IO.IsStdoutTTY())
			}
			if len(list.Items) == 0 {
				fmt.Fprintln(f.IO.ErrOut, "No posts found.")
				return nil
			}
			rows := make([][]string, 0, len(list.Items))
			for _, p := range list.Items {
				rows = append(rows, []string{p.QID, p.CreatedAt.Format("2006-01-02 15:04"), p.Title})
			}
			printTable(f.IO.Out, []string{"QID", "CREATED", "TITLE"}, rows)
			fmt.Fprintf(f.IO.ErrOut, "Page %d of %d (%d posts)\n", list.Page, list.TotalPages, list.Total)
			return nil
		},
	}
}

func postsViewCommand(f *cliapp.Factory) *cli.Command {
	return &cli.Command{
		Name:      "view",
		Usage:     "View a post as markdown, HTML, or its URL",
		UsageText: "markpost posts view <qid> [--format raw|html|url]",
		ArgsUsage: "<qid>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "format", Value: "raw", Usage: "raw markdown (default), rendered html, or url"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() != 1 {
				return cliapp.FlagErrorf("view requires exactly one <qid> argument")
			}
			qid := c.Args().First()
			client, err := f.Client()
			if err != nil {
				return err
			}
			switch format := c.String("format"); format {
			case "raw":
				md, err := client.RawMarkdown(c.Context, qid)
				if err != nil {
					return err
				}
				fmt.Fprint(f.IO.Out, md)
				return nil
			case "html":
				html, err := client.PostHTML(c.Context, qid)
				if err != nil {
					return err
				}
				fmt.Fprint(f.IO.Out, html)
				return nil
			case "url":
				fmt.Fprintln(f.IO.Out, postURL(client.BaseURL(), qid))
				return nil
			default:
				return cliapp.FlagErrorf("invalid --format %q: must be raw, html, or url", format)
			}
		},
	}
}

func postsDeleteCommand(f *cliapp.Factory) *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Aliases:   []string{"rm"},
		Usage:     "Delete one of your posts",
		UsageText: "markpost posts delete <qid> [--yes]",
		ArgsUsage: "<qid>",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "yes", Aliases: []string{"y"}, Usage: "skip the confirmation prompt (required when not interactive)"},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() != 1 {
				return cliapp.FlagErrorf("delete requires exactly one <qid> argument")
			}
			qid := c.Args().First()
			if !c.Bool("yes") {
				if !f.IO.CanPrompt() {
					return cliapp.FlagErrorf("--yes is required to delete in non-interactive mode")
				}
				answer, err := promptLine(f.IO, "Delete post "+qid+"? [y/N] ")
				if err != nil {
					return err
				}
				answer = strings.ToLower(strings.TrimSpace(answer))
				if answer != "y" && answer != "yes" {
					return cliapp.FlagErrorf("deletion canceled")
				}
			}
			client, err := f.AuthenticatedClient()
			if err != nil {
				return err
			}
			return client.DeletePost(c.Context, qid)
		},
	}
}
