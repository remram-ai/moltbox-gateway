package localexec

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/remram-ai/moltbox-gateway/internal/command"
	appconfig "github.com/remram-ai/moltbox-gateway/internal/config"
	"github.com/remram-ai/moltbox-gateway/internal/docker"
	"github.com/remram-ai/moltbox-gateway/internal/orchestrator"
	"github.com/remram-ai/moltbox-gateway/internal/secrets"
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
	if result.Route.Kind == cli.KindBootstrap || (result.Route.Kind == cli.KindGateway && result.Route.Action == "update") {
		cfg, err := appconfig.Load(r.configPath)
		if err != nil {
			payload, marshalErr := marshal(cli.Error(
				result.Route,
				localControlConfigErrorType(result.Route),
				fmt.Sprintf("failed to load gateway config from %s", r.configPath),
				err.Error(),
			))
			if marshalErr != nil {
				return nil, cli.ExitFailure, marshalErr
			}
			return payload, cli.ExitFailure, nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		manager := orchestrator.NewManager(
			cfg,
			docker.NewClient(cli.DockerSocketPath()),
			command.NewExecRunner(),
			secrets.NewHandler(cfg.Paths.SecretsRoot),
		)
		localControlResult, err := executeLocalControl(ctx, manager, result.Route)
		if err != nil {
			payload, marshalErr := marshal(cli.Error(
				result.Route,
				localControlFailureType(result.Route),
				localControlFailureMessage(result.Route),
				err.Error(),
			))
			if marshalErr != nil {
				return nil, cli.ExitFailure, marshalErr
			}
			return payload, cli.ExitFailure, nil
		}
		payload, err := marshal(localControlResult)
		return payload, cli.ExitCodeFromPayload(payload), err
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

func executeLocalControl(ctx context.Context, manager *orchestrator.Manager, route *cli.Route) (any, error) {
	switch {
	case route.Kind == cli.KindBootstrap:
		return manager.BootstrapGateway(ctx, route)
	case route.Kind == cli.KindGateway && route.Action == "update":
		return manager.GatewayUpdate(ctx, route)
	default:
		return nil, fmt.Errorf("unsupported local control route %s %s", route.Kind, route.Action)
	}
}

func localControlConfigErrorType(route *cli.Route) string {
	switch {
	case route.Kind == cli.KindBootstrap:
		return "bootstrap_config_failed"
	case route.Kind == cli.KindGateway && route.Action == "update":
		return "gateway_update_config_failed"
	default:
		return "local_control_config_failed"
	}
}

func localControlFailureType(route *cli.Route) string {
	switch {
	case route.Kind == cli.KindBootstrap:
		return "bootstrap_gateway_failed"
	case route.Kind == cli.KindGateway && route.Action == "update":
		return "gateway_update_failed"
	default:
		return "local_control_failed"
	}
}

func localControlFailureMessage(route *cli.Route) string {
	switch {
	case route.Kind == cli.KindBootstrap:
		return "failed to bootstrap the gateway control plane"
	case route.Kind == cli.KindGateway && route.Action == "update":
		return "failed to update the gateway control plane"
	default:
		return "failed to execute the local control action"
	}
}
