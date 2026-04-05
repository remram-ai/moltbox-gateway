package localexec

import (
	"bytes"
	"fmt"

	"github.com/remram-ai/moltbox-gateway/internal/client"
	"github.com/remram-ai/moltbox-gateway/pkg/cli"
)

type Runner struct {
	configPath string
	gatewayURL string
}

func New(configPath, gatewayURL string) *Runner {
	return &Runner{
		configPath: configPath,
		gatewayURL: gatewayURL,
	}
}

func (r *Runner) ExecuteArgs(args []string, secretValue string) ([]byte, int, error) {
	return r.ExecuteParse(cli.Parse(args), secretValue)
}

func (r *Runner) ExecuteParse(result cli.ParseResult, secretValue string) ([]byte, int, error) {
	switch {
	case result.Help:
		return []byte(helpPayload(result.HelpTopic)), cli.ExitOK, nil
	case result.Version:
		return []byte(fmt.Sprintf("moltbox %s\n", cli.Version)), cli.ExitOK, nil
	case result.Envelope != nil:
		payload, err := marshal(result.Envelope)
		return payload, result.Code, err
	}

	if result.Route == nil {
		payload, err := marshal(cli.Error(nil, "parse_error", "missing route", "use a documented moltbox command"))
		return payload, cli.ExitParseError, err
	}
	if result.Route.Kind == cli.KindBootstrap {
		payload, err := marshal(cli.NotImplemented(
			result.Route,
			"bootstrap gateway is not implemented yet",
			"bootstrap the gateway with the documented host runbook or implement the local bootstrap helper first",
		))
		return payload, cli.ExitNotImplemented, err
	}

	payload, err := client.NewHTTPClient(r.gatewayURL).Execute(result.Route, secretValue)
	if err != nil {
		payload, marshalErr := marshal(cli.Error(
			result.Route,
			"gateway_unreachable",
			fmt.Sprintf("failed to contact gateway at %s", r.gatewayURL),
			"verify the gateway container is running and the localhost control port is reachable",
		))
		if marshalErr != nil {
			return nil, cli.ExitFailure, marshalErr
		}
		return payload, cli.ExitFailure, nil
	}
	return payload, cli.ExitCodeFromPayload(payload), nil
}

func marshal(payload any) ([]byte, error) {
	var buffer bytes.Buffer
	if err := cli.WriteJSON(&buffer, payload); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func helpPayload(topic string) string {
	var buffer bytes.Buffer
	_ = cli.WriteHelp(&buffer, topic)
	return buffer.String()
}
