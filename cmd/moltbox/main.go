package main

import (
	"io"
	"os"

	"github.com/remram-ai/moltbox-gateway/internal/gateway"
	"github.com/remram-ai/moltbox-gateway/internal/runtime"
	"github.com/remram-ai/moltbox-gateway/internal/services"
	"github.com/remram-ai/moltbox-gateway/pkg/cli"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, _ io.Writer) int {
	result := cli.Parse(args)

	switch {
	case result.Help:
		_ = cli.WriteHelp(stdout)
		return cli.ExitOK
	case result.Version:
		_ = cli.WriteVersion(stdout)
		return cli.ExitOK
	case result.Envelope != nil:
		_ = cli.WriteJSON(stdout, result.Envelope)
		return result.Code
	}

	switch result.Route.Resource {
	case "gateway":
		return gateway.Handle(result.Route, stdout)
	case "dev", "test", "prod":
		return runtime.Handle(result.Route, stdout)
	case "ollama", "opensearch", "caddy":
		return services.Handle(result.Route, stdout)
	default:
		_ = cli.WriteJSON(stdout, cli.Error(result.Route, "parse_error", "unknown route target", "use a documented command"))
		return cli.ExitParseError
	}
}
