package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	appconfig "github.com/remram-ai/moltbox-gateway/internal/config"
	"github.com/remram-ai/moltbox-gateway/internal/localexec"
	"github.com/remram-ai/moltbox-gateway/internal/mcpstdio"
	"github.com/remram-ai/moltbox-gateway/pkg/cli"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	result := cli.Parse(args)

	if result.Route != nil && result.Route.Kind == cli.KindGatewayMCP {
		executor := localexec.New(appconfig.ConfigPath(), cli.GatewayURL())
		if err := mcpstdio.New(executor).Run(os.Stdin, stdout); err != nil {
			_, _ = fmt.Fprintf(stderr, "mcp stdio server failed: %v\n", err)
			return cli.ExitFailure
		}
		return cli.ExitOK
	}

	secretValue := ""
	if result.Route != nil && result.Route.Kind == cli.KindScopedSecrets {
		if result.Route.Action == "set" {
			if len(result.Route.NativeArgs) > 0 {
				secretValue = result.Route.NativeArgs[0]
			} else {
				var err error
				secretValue, err = loadSecretValue(os.Stdin)
				if err != nil {
					_ = cli.WriteJSON(stdout, cli.Error(
						result.Route,
						"secret_input_missing",
						"failed to read secret input",
						err.Error(),
					))
					return cli.ExitFailure
				}
			}
		}
	}

	payload, code, err := localexec.New(appconfig.ConfigPath(), cli.GatewayURL()).ExecuteParse(result, secretValue)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "moltbox execution failed: %v\n", err)
		return cli.ExitFailure
	}
	if _, err := stdout.Write(payload); err != nil {
		return cli.ExitFailure
	}
	return code
}

func loadSecretValue(stdin io.Reader) (string, error) {
	if value, ok := os.LookupEnv("MOLTBOX_SECRET_VALUE"); ok && value != "" {
		return value, nil
	}

	reader := bufio.NewReader(stdin)
	data, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	value := strings.TrimRight(data, "\r\n")
	if value == "" {
		return "", fmt.Errorf("pipe the secret value on stdin or set MOLTBOX_SECRET_VALUE")
	}
	return value, nil
}
