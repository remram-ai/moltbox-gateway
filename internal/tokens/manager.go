package tokens

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/remram-ai/moltbox-gateway/internal/secrets"
	"github.com/remram-ai/moltbox-gateway/pkg/cli"
)

const (
	secretScope  = "service"
	secretPrefix = "MCP_TOKEN_"
)

type ValidationResult struct {
	Authorized bool
	Name       string
}

type cachedToken struct {
	Name   string
	Digest [sha256.Size]byte
}

type Manager struct {
	store *secrets.Store

	mu          sync.RWMutex
	cache       map[string]cachedToken
	cacheLoaded bool
}

func NewManager(root string) *Manager {
	return &Manager{
		store: secrets.NewStore(root),
		cache: make(map[string]cachedToken),
	}
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
	m.putCachedToken(displayName, token)
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
	m.putCachedToken(displayName, token)
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
	m.deleteCachedToken(displayName)
	return cli.GatewayTokenDeleteResult{
		OK:      true,
		Route:   route,
		Name:    displayName,
		Deleted: deleted,
	}, nil
}

func (m *Manager) List(route *cli.Route) (cli.GatewayTokenListResult, error) {
	entries, err := m.cachedTokens()
	if err != nil {
		return cli.GatewayTokenListResult{}, err
	}
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	slices.Sort(names)

	result := cli.GatewayTokenListResult{
		OK:     true,
		Route:  route,
		Tokens: make([]cli.GatewayTokenInfo, 0, len(names)),
	}
	for _, name := range names {
		result.Tokens = append(result.Tokens, cli.GatewayTokenInfo{Name: name})
	}
	return result, nil
}

func (m *Manager) ValidateBearerToken(header string) (ValidationResult, error) {
	token, ok := bearerToken(header)
	if !ok {
		return ValidationResult{}, nil
	}

	candidateDigest := digestToken(token)
	entries, err := m.cachedTokens()
	if err != nil {
		return ValidationResult{}, err
	}
	for _, entry := range entries {
		if subtle.ConstantTimeCompare(entry.Digest[:], candidateDigest[:]) == 1 {
			return ValidationResult{Authorized: true, Name: entry.Name}, nil
		}
	}
	return ValidationResult{}, nil
}

func (m *Manager) cachedTokens() (map[string]cachedToken, error) {
	m.mu.RLock()
	if m.cacheLoaded {
		snapshot := cloneCache(m.cache)
		m.mu.RUnlock()
		return snapshot, nil
	}
	m.mu.RUnlock()

	loaded, err := m.loadCache()
	if err != nil {
		return nil, err
	}
	return loaded, nil
}

func (m *Manager) loadCache() (map[string]cachedToken, error) {
	names, err := m.store.List(secretScope)
	if err != nil {
		return nil, err
	}

	loaded := make(map[string]cachedToken, len(names))
	for _, secretName := range names {
		displayName, ok := displayNameForSecret(secretName)
		if !ok {
			continue
		}
		stored, err := m.store.Get(secretScope, secretName)
		if err != nil {
			if err == secrets.ErrSecretNotFound {
				continue
			}
			return nil, err
		}
		loaded[displayName] = cachedToken{
			Name:   displayName,
			Digest: digestToken(stored),
		}
	}

	m.mu.Lock()
	m.cache = loaded
	m.cacheLoaded = true
	snapshot := cloneCache(m.cache)
	m.mu.Unlock()
	return snapshot, nil
}

func (m *Manager) putCachedToken(name, token string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cache == nil {
		m.cache = make(map[string]cachedToken)
	}
	m.cache[name] = cachedToken{Name: name, Digest: digestToken(token)}
	m.cacheLoaded = true
}

func (m *Manager) deleteCachedToken(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cache == nil {
		m.cache = make(map[string]cachedToken)
	}
	delete(m.cache, name)
	m.cacheLoaded = true
}

func cloneCache(source map[string]cachedToken) map[string]cachedToken {
	cloned := make(map[string]cachedToken, len(source))
	for name, entry := range source {
		cloned[name] = entry
	}
	return cloned
}

func digestToken(token string) [sha256.Size]byte {
	return sha256.Sum256([]byte(token))
}

func bearerToken(header string) (string, bool) {
	trimmed := strings.TrimSpace(header)
	if !strings.HasPrefix(trimmed, "Bearer ") {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(trimmed, "Bearer "))
	if token == "" {
		return "", false
	}
	return token, true
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
