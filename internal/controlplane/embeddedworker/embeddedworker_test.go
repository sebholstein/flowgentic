package embeddedworker

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	appconfig "github.com/sebastianm/flowgentic/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockConfigStore struct {
	secret    string
	getErr    error
	upsertErr error
}

func (m *mockConfigStore) GetSecret(_ context.Context) (string, error) {
	return m.secret, m.getErr
}

func (m *mockConfigStore) UpsertSecret(_ context.Context, secret string) error {
	m.secret = secret
	return m.upsertErr
}

func TestResolveSecret(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("returns file config secret when set", func(t *testing.T) {
		store := &mockConfigStore{}
		fileCfg := appconfig.EmbeddedWorkerConfig{Secret: "file-secret"}

		result := resolveSecret(context.Background(), log, fileCfg, store)

		assert.Equal(t, "file-secret", result)
	})

	t.Run("returns database secret when file config is empty", func(t *testing.T) {
		store := &mockConfigStore{secret: "db-secret"}
		fileCfg := appconfig.EmbeddedWorkerConfig{}

		result := resolveSecret(context.Background(), log, fileCfg, store)

		assert.Equal(t, "db-secret", result)
	})

	t.Run("generates new secret when database is empty", func(t *testing.T) {
		store := &mockConfigStore{secret: ""}
		fileCfg := appconfig.EmbeddedWorkerConfig{}

		result := resolveSecret(context.Background(), log, fileCfg, store)

		assert.NotEmpty(t, result)
		assert.Len(t, result, 64)
		assert.Equal(t, result, store.secret)
	})

	t.Run("generates new secret when database returns error", func(t *testing.T) {
		store := &mockConfigStore{getErr: errors.New("db error")}
		fileCfg := appconfig.EmbeddedWorkerConfig{}

		result := resolveSecret(context.Background(), log, fileCfg, store)

		assert.NotEmpty(t, result)
		assert.Len(t, result, 64)
	})

	t.Run("persists generated secret on first run", func(t *testing.T) {
		store := &mockConfigStore{secret: ""}
		fileCfg := appconfig.EmbeddedWorkerConfig{}

		result := resolveSecret(context.Background(), log, fileCfg, store)

		require.NotEmpty(t, result)
		assert.Equal(t, result, store.secret)
	})

	t.Run("returns generated secret even when upsert fails", func(t *testing.T) {
		store := &mockConfigStore{secret: "", upsertErr: errors.New("upsert error")}
		fileCfg := appconfig.EmbeddedWorkerConfig{}

		result := resolveSecret(context.Background(), log, fileCfg, store)

		assert.NotEmpty(t, result)
		assert.Len(t, result, 64)
	})

	t.Run("file config secret takes priority over database secret", func(t *testing.T) {
		store := &mockConfigStore{secret: "db-secret"}
		fileCfg := appconfig.EmbeddedWorkerConfig{Secret: "file-secret"}

		result := resolveSecret(context.Background(), log, fileCfg, store)

		assert.Equal(t, "file-secret", result)
		assert.Equal(t, "db-secret", store.secret)
	})
}

func TestGenerateSecret(t *testing.T) {
	t.Run("generates 64 character hex string", func(t *testing.T) {
		secret := generateSecret()
		assert.Len(t, secret, 64)
	})

	t.Run("generates unique secrets", func(t *testing.T) {
		secret1 := generateSecret()
		secret2 := generateSecret()
		assert.NotEqual(t, secret1, secret2)
	})

	t.Run("generates valid hex string", func(t *testing.T) {
		secret := generateSecret()
		for _, c := range secret {
			assert.True(t, (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f'),
				"character %q is not valid hex", c)
		}
	})
}
