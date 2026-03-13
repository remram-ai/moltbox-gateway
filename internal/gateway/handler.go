package gateway

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/remram-ai/moltbox-gateway/internal/docker"
	"github.com/remram-ai/moltbox-gateway/pkg/cli"
)

func Handle(route *cli.Route, out io.Writer) int {
	switch route.Kind {
	case cli.KindGatewayDocker:
		return handleDockerPing(route, out)
	case cli.KindGateway:
		return writeNotImplemented(out, route, fmt.Sprintf("gateway %s is not implemented in phase 1", route.Action))
	case cli.KindGatewayService:
		return writeNotImplemented(out, route, fmt.Sprintf("gateway service %s %s is not implemented in phase 1", route.Action, route.Subject))
	default:
		_ = cli.WriteJSON(out, cli.Error(route, "parse_error", "unsupported gateway route", "use a documented gateway command"))
		return cli.ExitParseError
	}
}

func handleDockerPing(route *cli.Route, out io.Writer) int {
	client := docker.NewClient(cli.DockerSocketPath())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := client.Version(ctx)
	if err != nil {
		_ = cli.WriteJSON(out, cli.Error(
			route,
			"docker_unavailable",
			fmt.Sprintf("failed to contact Docker via %s", cli.DockerSocketPath()),
			"verify Docker is running and the current user can access the Docker socket",
		))
		return cli.ExitFailure
	}

	_ = cli.WriteJSON(out, cli.DockerPingResult{
		OK:            true,
		Route:         route,
		DockerVersion: info.Version,
		APIVersion:    info.APIVersion,
		MinAPIVersion: info.MinAPIVersion,
		GitCommit:     info.GitCommit,
		GoVersion:     info.GoVersion,
		OS:            info.OS,
		Arch:          info.Arch,
		KernelVersion: info.KernelVersion,
	})
	return cli.ExitOK
}

func writeNotImplemented(out io.Writer, route *cli.Route, message string) int {
	_ = cli.WriteJSON(out, cli.NotImplemented(
		route,
		message,
		"phase 1 only installs the routing scaffold and Docker connectivity check",
	))
	return cli.ExitNotImplemented
}
