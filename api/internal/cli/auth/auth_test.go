package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockKeyring implements the Keyring interface for testing.
type mockKeyring struct {
	data map[string]string
}

func newMockKeyring() *mockKeyring {
	return &mockKeyring{data: make(map[string]string)}
}

func (m *mockKeyring) Set(service, account, password string) error {
	m.data[service+"/"+account] = password
	return nil
}

func (m *mockKeyring) Get(service, account string) (string, error) {
	v, ok := m.data[service+"/"+account]
	if !ok {
		return "", fmt.Errorf("not found")
	}
	return v, nil
}

func (m *mockKeyring) Delete(service, account string) error {
	delete(m.data, service+"/"+account)
	return nil
}

func TestCredentialsIsEmpty(t *testing.T) {
	tests := []struct {
		name   string
		creds  Credentials
		expect bool
	}{
		{"empty", Credentials{}, true},
		{"with api key", Credentials{APIKey: "key"}, false},
		{"with token", Credentials{Token: "tok"}, false},
		{"with both", Credentials{APIKey: "key", Token: "tok"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, tt.creds.IsEmpty())
		})
	}
}

func TestCredentialsAuthHeader(t *testing.T) {
	tests := []struct {
		name    string
		creds   Credentials
		wantKey string
		wantVal string
	}{
		{"empty", Credentials{}, "", ""},
		{"api key", Credentials{APIKey: "test-key"}, "X-API-Key", "test-key"},
		{"token", Credentials{Token: "jwt-token"}, "Authorization", "Bearer jwt-token"},
		{"both prefers token", Credentials{APIKey: "key", Token: "tok"}, "Authorization", "Bearer tok"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, val := tt.creds.AuthHeader()
			assert.Equal(t, tt.wantKey, key)
			assert.Equal(t, tt.wantVal, val)
		})
	}
}

func TestStoreSaveAndLoad(t *testing.T) {
	kr := newMockKeyring()
	store := NewStore(kr)

	creds := &Credentials{APIKey: "test-api-key-123"}
	require.NoError(t, store.Save(creds))

	loaded, err := store.Load()
	require.NoError(t, err)
	assert.Equal(t, "test-api-key-123", loaded.APIKey)
}

func TestStoreLoadEmpty(t *testing.T) {
	kr := newMockKeyring()
	store := NewStore(kr)

	creds, err := store.Load()
	require.NoError(t, err)
	assert.True(t, creds.IsEmpty())
}

func TestStoreClear(t *testing.T) {
	kr := newMockKeyring()
	store := NewStore(kr)

	require.NoError(t, store.Save(&Credentials{APIKey: "key"}))
	require.NoError(t, store.Clear())

	creds, err := store.Load()
	require.NoError(t, err)
	assert.True(t, creds.IsEmpty())
}

func TestStoreClearNoExisting(t *testing.T) {
	kr := newMockKeyring()
	store := NewStore(kr)

	// Clear when nothing stored should not error.
	assert.NoError(t, store.Clear())
}

func TestStoreSaveToken(t *testing.T) {
	kr := newMockKeyring()
	store := NewStore(kr)

	creds := &Credentials{Token: "jwt-test-token"}
	require.NoError(t, store.Save(creds))

	loaded, err := store.Load()
	require.NoError(t, err)
	assert.Equal(t, "jwt-test-token", loaded.Token)
	assert.Empty(t, loaded.APIKey)
}

func TestStoreOverwrite(t *testing.T) {
	kr := newMockKeyring()
	store := NewStore(kr)

	require.NoError(t, store.Save(&Credentials{APIKey: "first"}))
	require.NoError(t, store.Save(&Credentials{APIKey: "second"}))

	loaded, err := store.Load()
	require.NoError(t, err)
	assert.Equal(t, "second", loaded.APIKey)
}

func TestStoreLoadInvalidJSON(t *testing.T) {
	kr := newMockKeyring()
	kr.data[KeyringService+"/"+KeyringAccount] = "not-json"

	store := NewStore(kr)
	creds, err := store.Load()
	require.NoError(t, err)
	assert.True(t, creds.IsEmpty())
}

func TestValidateCredentialsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/", r.URL.Path)
		assert.Equal(t, "test-key", r.Header.Get("X-API-Key"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	err := ValidateCredentials(server.URL, &Credentials{APIKey: "test-key"})
	assert.NoError(t, err)
}

func TestValidateCredentialsFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	err := ValidateCredentials(server.URL, &Credentials{APIKey: "bad-key"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "authentication failed")
}

func TestValidateCredentialsEmpty(t *testing.T) {
	err := ValidateCredentials("http://localhost", &Credentials{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no credentials provided")
}

func TestValidateCredentialsJWT(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer jwt-test", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := ValidateCredentials(server.URL, &Credentials{Token: "jwt-test"})
	assert.NoError(t, err)
}

func TestValidateCredentialsConnectionError(t *testing.T) {
	err := ValidateCredentials("http://127.0.0.1:1", &Credentials{APIKey: "key"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connecting to API")
}
