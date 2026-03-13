package secrets

import (
	"errors"
	"fmt"

	"github.com/remram-ai/moltbox-gateway/pkg/cli"
)

type Handler struct {
	store *Store
}

func NewHandler(root string) *Handler {
	return &Handler{store: NewStore(root)}
}

func (h *Handler) Execute(route *cli.Route, value string) any {
	scope := route.Resource

	switch route.Action {
	case "set":
		if err := h.store.Set(scope, route.Subject, value); err != nil {
			return cli.Error(route, "secret_store_failed", "failed to store secret", err.Error())
		}
		return cli.SecretSetResult{
			OK:     true,
			Route:  route,
			Scope:  scope,
			Name:   route.Subject,
			Stored: true,
		}
	case "list":
		names, err := h.store.List(scope)
		if err != nil {
			return cli.Error(route, "secret_list_failed", "failed to list secrets", err.Error())
		}

		items := make([]cli.SecretListItem, 0, len(names))
		for _, name := range names {
			items = append(items, cli.SecretListItem{Scope: scope, Name: name})
		}
		return cli.SecretListResult{
			OK:      true,
			Route:   route,
			Scope:   scope,
			Secrets: items,
		}
	case "delete":
		deleted, err := h.store.Delete(scope, route.Subject)
		if err != nil {
			return cli.Error(route, "secret_delete_failed", "failed to delete secret", err.Error())
		}
		return cli.SecretDeleteResult{
			OK:      true,
			Route:   route,
			Scope:   scope,
			Name:    route.Subject,
			Deleted: deleted,
		}
	default:
		return cli.Error(route, "parse_error", fmt.Sprintf("unsupported secrets action '%s'", route.Action), "use set, list, or delete")
	}
}

func (h *Handler) Get(scope, name string) (string, error) {
	value, err := h.store.Get(scope, name)
	if errors.Is(err, ErrSecretNotFound) {
		return "", nil
	}
	return value, err
}

func (h *Handler) Resolve(scope string, names []string) (map[string]string, error) {
	return h.store.Resolve(scope, names)
}
