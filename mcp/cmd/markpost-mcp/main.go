// Command markpost-mcp is the MCP server exposing a markpost instance to AI
// agents. It is a standalone Go binary distributed separately from the
// markpost image; point it at any instance with --url and credentials in the
// environment.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/urfave/cli/v2"

	"github.com/jukanntenn/markpost/mcp/internal/config"
	"github.com/jukanntenn/markpost/mcp/internal/server"
)

// version is injected at build time (-ldflags "-X main.version=...").
var version = "dev"

func main() {
	log.SetFlags(0)

	app := &cli.App{
		Name:     "markpost-mcp",
		Usage:    "MCP server exposing a markpost instance to AI agents",
		Version:  version,
		Commands: []*cli.Command{stdioCommand(), httpCommand()},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

// commonFlags are the shared runtime flags. urfave/cli v2 does not propagate
// app-level flags to subcommands, so both commands carry them directly.
func commonFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "url",
			Usage:   "markpost instance base URL (scheme://host[:port])",
			EnvVars: []string{"MARKPOST_MCP_URL"},
		},
		&cli.StringFlag{
			Name:    "toolsets",
			Usage:   "comma-separated toolsets to enable (posts, delivery, account, admin, all)",
			EnvVars: []string{"MARKPOST_MCP_TOOLSETS"},
			Value:   config.DefaultToolsets,
		},
		&cli.BoolFlag{
			Name:    "read-only",
			Usage:   "disable every tool that writes",
			EnvVars: []string{"MARKPOST_MCP_READ_ONLY"},
		},
	}
}

func stdioCommand() *cli.Command {
	return &cli.Command{
		Name:  "stdio",
		Usage: "start the MCP server on stdin/stdout (for local MCP hosts)",
		Flags: commonFlags(),
		Action: func(cctx *cli.Context) error {
			cfg, err := resolveConfig(cctx)
			if err != nil {
				return err
			}
			ctx := signalContext()
			srv, err := server.New(ctx, server.Options{Config: cfg})
			if err != nil {
				return fmt.Errorf("startup: %w", err)
			}
			return srv.Run(ctx, &mcp.StdioTransport{})
		},
	}
}

func httpCommand() *cli.Command {
	return &cli.Command{
		Name:  "http",
		Usage: "start the MCP server as a streamable-http endpoint (for remote MCP hosts)",
		Flags: append(commonFlags(),
			&cli.StringFlag{
				Name:    "addr",
				Usage:   "listen address",
				EnvVars: []string{"MARKPOST_MCP_HTTP_ADDR"},
				Value:   "127.0.0.1:8973",
			},
			&cli.StringFlag{
				Name:    "path",
				Usage:   "endpoint path",
				EnvVars: []string{"MARKPOST_MCP_HTTP_PATH"},
				Value:   "/mcp",
			},
			&cli.StringFlag{
				Name:    "http-token",
				Usage:   "bearer token MCP clients must present (empty = no auth; bind loopback)",
				EnvVars: []string{"MARKPOST_MCP_HTTP_TOKEN"},
			},
		),
		Action: func(cctx *cli.Context) error {
			cfg, err := resolveConfig(cctx)
			if err != nil {
				return err
			}
			cfg.HTTPAddr = cctx.String("addr")
			cfg.HTTPPath = cctx.String("path")
			cfg.HTTPToken = cctx.String("http-token")

			// One server instance shared across requests: stateless
			// streamable-http has no session state to protect.
			srv, err := server.New(signalContext(), server.Options{Config: cfg})
			if err != nil {
				return fmt.Errorf("startup: %w", err)
			}

			mux := http.NewServeMux()
			mux.Handle(cfg.HTTPPath, server.HTTPHandler(func(*http.Request) *mcp.Server { return srv }, cfg.HTTPToken))
			hs := &http.Server{Addr: cfg.HTTPAddr, Handler: mux}

			errCh := make(chan error, 1)
			go func() { errCh <- hs.ListenAndServe() }()
			log.Printf("markpost-mcp http listening on %s%s", cfg.HTTPAddr, cfg.HTTPPath)

			ctx := signalContext()
			select {
			case err := <-errCh:
				return err
			case <-ctx.Done():
				return hs.Shutdown(context.Background())
			}
		},
	}
}

// resolveConfig assembles the runtime config from flags + environment,
// failing fast on anything missing.
func resolveConfig(cctx *cli.Context) (*config.Config, error) {
	cfg := &config.Config{
		BaseURL:  cctx.String("url"),
		Toolsets: config.ParseToolsets(cctx.String("toolsets")),
		ReadOnly: cctx.Bool("read-only"),
		Version:  version,
	}
	cfg.Username, cfg.Password = config.CredentialsFromEnv()

	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("missing markpost instance URL: set --url or MARKPOST_MCP_URL")
	}
	if len(cfg.Toolsets) == 0 {
		return nil, fmt.Errorf("no toolsets enabled: --toolsets is empty")
	}
	if err := cfg.Credentials(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// signalContext returns a context cancelled on SIGINT/SIGTERM.
func signalContext() context.Context {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	_ = stop // stop stays alive for the process lifetime
	return ctx
}
