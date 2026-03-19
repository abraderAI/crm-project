package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClerkClient_EmptyKey(t *testing.T) {
	c := NewClerkClient("")
	assert.Nil(t, c, "empty key should return nil")
}

func TestNewClerkClient_NonEmptyKey(t *testing.T) {
	c := NewClerkClient("sk_test_abc")
	require.NotNil(t, c)
	assert.Equal(t, "https://api.clerk.com", c.baseURL)
}

func TestClerkClient_GetUser_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Contains(t, r.URL.Path, "user_abc")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"email_addresses": [{"email_address": "alice@example.com"}],
			"first_name": "Alice",
			"last_name": "Smith"
		}`))
	}))
	defer ts.Close()

	c := NewClerkClientForTest("test-key", ts.URL)

	user, err := c.GetUser(context.Background(), "user_abc")
	require.NoError(t, err)
	assert.Equal(t, "alice@example.com", user.Email)
	assert.Equal(t, "Alice Smith", user.DisplayName)
}

func TestClerkClient_GetUser_UsernameWhenNoName(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"email_addresses": [{"email_address": "bob@example.com"}],
			"username": "bobsmith"
		}`))
	}))
	defer ts.Close()

	c := NewClerkClientForTest("key", ts.URL)

	user, err := c.GetUser(context.Background(), "user_bob")
	require.NoError(t, err)
	assert.Equal(t, "bob@example.com", user.Email)
	assert.Equal(t, "bobsmith", user.DisplayName)
}

func TestClerkClient_GetUser_NoEmailAddresses(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"first_name":"Jane","last_name":"Doe","email_addresses":[]}`))
	}))
	defer ts.Close()

	c := NewClerkClientForTest("key", ts.URL)

	user, err := c.GetUser(context.Background(), "user_jane")
	require.NoError(t, err)
	assert.Equal(t, "", user.Email)
	assert.Equal(t, "Jane Doe", user.DisplayName)
}

func TestClerkClient_GetUser_HTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	c := NewClerkClientForTest("bad", ts.URL)

	_, err := c.GetUser(context.Background(), "user_x")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestClerkClient_GetUser_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer ts.Close()

	c := NewClerkClientForTest("key", ts.URL)

	_, err := c.GetUser(context.Background(), "user_x")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing clerk response")
}
