package tokens

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/remram-ai/moltbox-gateway/internal/secrets"
	"github.com/remram-ai/moltbox-gateway/pkg/cli"
)

const (
	secretScope = "service"
	secretPrefix = "MCP_TOKEN_"
)

type Manager struct {
	store *secrets.Store
}

func NewManager(root string) *Manager {
	return &Manager{store: secrets.NewStore(root)}
}

func (m *Manager) Create(route *cli.Route) (cli.GatewayTokenCreateResult, error) {
	secretName, displayName, err := secretNameForRoute(route)
	if err != nil {
		return cli.GatewayTokenCreateResult{}, err
	}
	token, err := generateToken()
	if err != nil {
		return cli.GatewayTokenCreateResult{}, err
	}
	if err := m.store.Set(secretScope, secretName, token); err != nil {
		return cli.GatewayTokenCreateResult{}, err
	}
	return cli.GatewayTokenCreateResult{
		OK:      true,
		Route:   route,
		Name:    displayName,
		Token:   token,
		Created: true,
	}, nil
}

func (m *Manager) Rotate(route *cli.Route) (cli.GatewayTokenRotateResult, error) {
	secretName, displayName, err := secretNameForRoute(route)
	if err != nil {
		return cli.GatewayTokenRotateResult{}, err
	}
	token, err := generateToken()
	if err != nil {
		return cli.GatewayTokenRotateResult{}, err
	}
	if err := m.store.Set(secretScope, secretName, token); err != nil {
		return cli.GatewayTokenRotateResult{}, err
	}
	return cli.GatewayTokenRotateResult{
		OK:      true,
		Route:   route,
		Name:    displayName,
		Token:   token,
		Rotated: true,
	}, nil
}

func (m *Manager) Delete(route *cli.Route) (cli.GatewayTokenDeleteResult, error) {
	secretName, displayName, err := secretNameForRoute(route)
	if err != nil {
		return cli.GatewayTokenDeleteResult{}, err
	}
	deleted, err := m.store.Delete(secretScope, secretName)
	if err != nil {
		return cli.GatewayTokenDeleteResult{}, err
	}
	return cli.GatewayTokenDeleteResult{
		OK:      true,
		Route:   route,
		Name:    displayName,
		Deleted: deleted,
	}, nil
}

func (m *Manager) List(route *cli.Route) (cli.GatewayTokenListResult, error) {
	names, err := m.store.List(secretScope)
	if err != nil {
		return cli.GatewayTokenListResult{}, err
	}
	result := cli.GatewayTokenListResult{
		OK:     true,
		Route:  route,
		Tokens: make([]cli.GatewayTokenInfo, 0, len(names)),
	}
	for _, name := range names {
		displayName, ok := displayNameForSecret(name)
		if !ok {
			continue
		}
		result.Tokens = append(result.Tokens, cli.GatewayTokenInfo{Name: displayName})
	}
	return result, nil
}

func (m *Manager) ValidateBearerToken(header string) (bool, error) {
	trimmed := strings.TrimSpace(header)
	if !strings.HasPrefix(trimmed, "Bearer ") {
		return false, nil
	}
	token := strings.TrimSpace(strings.TrimPrefix(trimmed, "Bearer "))
	if token == "" {
		return false, nil
	}

	names, err := m.store.List(secretScope)
	if err != nil {
		return false, err
	}
	for _, name := range names {
		if _, ok := displayNameForSecret(name); !ok {
			continue
		}
		stored, err := m.store.Get(secretScope, name)
		if err != nil {
			if err == secrets.ErrSecretNotFound {
				continue
			}
			return false, err
		}
		if stored == token {
			return true, nil
		}
	}
	return false, nil
}

func generateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func secretNameForRoute(route *cli.Route) (string, string, error) {
	if route == nil {
		return "", "", fmt.Errorf("missing token route")
	}
	displayName := strings.TrimSpace(route.Subject)
	if displayName == "" {
		return "", "", fmt.Errorf("missing token name")
	}
	normalized, err := normalizeTokenName(displayName)
	if err != nil {
		return "", "", err
	}
	return secretPrefix + normalized, normalized, nil
}

func displayNameForSecret(secretName string) (string, bool) {
	if !strings.HasPrefix(secretName, secretPrefix) {
		return "", false
	}
	name := strings.TrimPrefix(secretName, secretPrefix)
	if name == "" {
		return "", false
	}
	return name, true
}

func normalizeTokenName(name string) (string, error) {
	normalized, err := secrets.NormalizeName(name)
	if err != nil {
		return "", fmt.Errorf("invalid token name %q", name)
	}
	return normalized, nil
}
