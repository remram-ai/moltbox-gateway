package services

import (
	"fmt"
	"io"
	"strings"

	"github.com/remram-ai/moltbox-gateway/pkg/cli"
)

func Handle(route *cli.Route, out io.Writer) int {
	if route.Kind != cli.KindServiceNative {
		_ = cli.WriteJSON(out, cli.Error(route, "parse_error", "unsupported service route", "use a documented native service command"))
		return cli.ExitParseError
	}

	_ = cli.WriteJSON(out, cli.NotImplemented(
		route,
		fmt.Sprintf("%s passthrough is not implemented in phase 1 (requested: %s)", route.Resource, strings.Join(route.NativeArgs, " ")),
		"phase 2 will add managed service passthrough execution",
	))
	return cli.ExitNotImplemented
}
