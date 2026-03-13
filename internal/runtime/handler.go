package runtime

import (
	"fmt"
	"io"
	"strings"

	"github.com/remram-ai/moltbox-gateway/pkg/cli"
)

func Handle(route *cli.Route, out io.Writer) int {
	var message string

	switch route.Kind {
	case cli.KindRuntimeAction:
		message = fmt.Sprintf("%s %s is not implemented in phase 1", route.Environment, route.Action)
	case cli.KindRuntimeNative:
		message = fmt.Sprintf(
			"%s openclaw passthrough is not implemented in phase 1 (requested: %s)",
			route.Environment,
			strings.Join(route.NativeArgs, " "),
		)
	default:
		_ = cli.WriteJSON(out, cli.Error(route, "parse_error", "unsupported runtime route", "use a documented environment command"))
		return cli.ExitParseError
	}

	_ = cli.WriteJSON(out, cli.NotImplemented(
		route,
		message,
		"phase 2 will add runtime orchestration and native passthrough execution",
	))
	return cli.ExitNotImplemented
}
