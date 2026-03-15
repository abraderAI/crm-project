// Package client provides a typed Go HTTP client wrapping the CRM REST API endpoints.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client is a typed HTTP client for the CRM REST API.
type Client struct {
	baseURL    string
	httpClient *http.Client
	authHeader string // Header name (e.g. "X-API-Key" or "Authorization").
	authValue  string // Header value.
}

// New creates a new API client.
func New(baseURL, authHeader, authValue string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		authHeader: authHeader,
		authValue:  authValue,
	}
}

// NewWithHTTPClient creates a new API client with a custom http.Client (for testing).
func NewWithHTTPClient(baseURL, authHeader, authValue string, hc *http.Client) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: hc,
		authHeader: authHeader,
		authValue:  authValue,
	}
}

// APIError represents an RFC 7807 Problem Details error from the API.
type APIError struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s: %s (HTTP %d)", e.Title, e.Detail, e.Status)
	}
	return fmt.Sprintf("%s (HTTP %d)", e.Title, e.Status)
}

// PageInfo holds pagination metadata from the API.
type PageInfo struct {
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// ListResponse is a generic paginated list response.
type ListResponse struct {
	Data     json.RawMessage `json:"data"`
	PageInfo *PageInfo       `json:"page_info"`
}

// ListParams holds pagination and filtering parameters for list operations.
type ListParams struct {
	Cursor string
	Limit  int
	Query  map[string]string // Additional query parameters.
}

// Entity is a generic map representing any API entity (org, space, board, thread, message).
type Entity = map[string]any

// doRequest executes an HTTP request with auth headers and returns the response body.
func (c *Client) doRequest(method, path string, body any) ([]byte, int, error) {
	u := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, u, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.authHeader != "" && c.authValue != "" {
		req.Header.Set(c.authHeader, c.authValue)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr APIError
		if json.Unmarshal(data, &apiErr) == nil && apiErr.Title != "" {
			apiErr.Status = resp.StatusCode
			return nil, resp.StatusCode, &apiErr
		}
		return nil, resp.StatusCode, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(data))
	}

	return data, resp.StatusCode, nil
}

// buildListURL constructs a URL with pagination and query parameters.
func buildListURL(path string, params *ListParams) string {
	if params == nil {
		return path
	}
	v := url.Values{}
	if params.Cursor != "" {
		v.Set("cursor", params.Cursor)
	}
	if params.Limit > 0 {
		v.Set("limit", fmt.Sprintf("%d", params.Limit))
	}
	for k, val := range params.Query {
		v.Set(k, val)
	}
	if len(v) > 0 {
		return path + "?" + v.Encode()
	}
	return path
}

// --- Orgs ---

// ListOrgs returns a paginated list of orgs.
func (c *Client) ListOrgs(params *ListParams) ([]Entity, *PageInfo, error) {
	return c.listEntities(buildListURL("/v1/orgs", params))
}

// GetOrg returns a single org by ID or slug.
func (c *Client) GetOrg(orgRef string) (Entity, error) {
	return c.getEntity(fmt.Sprintf("/v1/orgs/%s", orgRef))
}

// CreateOrg creates a new org.
func (c *Client) CreateOrg(body map[string]any) (Entity, error) {
	return c.createEntity("/v1/orgs", body)
}

// UpdateOrg updates an existing org.
func (c *Client) UpdateOrg(orgRef string, body map[string]any) (Entity, error) {
	return c.updateEntity(fmt.Sprintf("/v1/orgs/%s", orgRef), body)
}

// --- Spaces ---

// ListSpaces returns a paginated list of spaces within an org.
func (c *Client) ListSpaces(orgRef string, params *ListParams) ([]Entity, *PageInfo, error) {
	return c.listEntities(buildListURL(fmt.Sprintf("/v1/orgs/%s/spaces", orgRef), params))
}

// GetSpace returns a single space.
func (c *Client) GetSpace(orgRef, spaceRef string) (Entity, error) {
	return c.getEntity(fmt.Sprintf("/v1/orgs/%s/spaces/%s", orgRef, spaceRef))
}

// CreateSpace creates a new space.
func (c *Client) CreateSpace(orgRef string, body map[string]any) (Entity, error) {
	return c.createEntity(fmt.Sprintf("/v1/orgs/%s/spaces", orgRef), body)
}

// UpdateSpace updates an existing space.
func (c *Client) UpdateSpace(orgRef, spaceRef string, body map[string]any) (Entity, error) {
	return c.updateEntity(fmt.Sprintf("/v1/orgs/%s/spaces/%s", orgRef, spaceRef), body)
}

// --- Boards ---

// ListBoards returns a paginated list of boards within a space.
func (c *Client) ListBoards(orgRef, spaceRef string, params *ListParams) ([]Entity, *PageInfo, error) {
	return c.listEntities(buildListURL(fmt.Sprintf("/v1/orgs/%s/spaces/%s/boards", orgRef, spaceRef), params))
}

// GetBoard returns a single board.
func (c *Client) GetBoard(orgRef, spaceRef, boardRef string) (Entity, error) {
	return c.getEntity(fmt.Sprintf("/v1/orgs/%s/spaces/%s/boards/%s", orgRef, spaceRef, boardRef))
}

// CreateBoard creates a new board.
func (c *Client) CreateBoard(orgRef, spaceRef string, body map[string]any) (Entity, error) {
	return c.createEntity(fmt.Sprintf("/v1/orgs/%s/spaces/%s/boards", orgRef, spaceRef), body)
}

// UpdateBoard updates an existing board.
func (c *Client) UpdateBoard(orgRef, spaceRef, boardRef string, body map[string]any) (Entity, error) {
	return c.updateEntity(fmt.Sprintf("/v1/orgs/%s/spaces/%s/boards/%s", orgRef, spaceRef, boardRef), body)
}

// --- Threads (Leads) ---

// ListThreads returns a paginated list of threads within a board.
func (c *Client) ListThreads(orgRef, spaceRef, boardRef string, params *ListParams) ([]Entity, *PageInfo, error) {
	return c.listEntities(buildListURL(fmt.Sprintf("/v1/orgs/%s/spaces/%s/boards/%s/threads", orgRef, spaceRef, boardRef), params))
}

// GetThread returns a single thread.
func (c *Client) GetThread(orgRef, spaceRef, boardRef, threadRef string) (Entity, error) {
	return c.getEntity(fmt.Sprintf("/v1/orgs/%s/spaces/%s/boards/%s/threads/%s", orgRef, spaceRef, boardRef, threadRef))
}

// CreateThread creates a new thread.
func (c *Client) CreateThread(orgRef, spaceRef, boardRef string, body map[string]any) (Entity, error) {
	return c.createEntity(fmt.Sprintf("/v1/orgs/%s/spaces/%s/boards/%s/threads", orgRef, spaceRef, boardRef), body)
}

// UpdateThread updates an existing thread.
func (c *Client) UpdateThread(orgRef, spaceRef, boardRef, threadRef string, body map[string]any) (Entity, error) {
	return c.updateEntity(fmt.Sprintf("/v1/orgs/%s/spaces/%s/boards/%s/threads/%s", orgRef, spaceRef, boardRef, threadRef), body)
}

// --- Messages ---

// ListMessages returns a paginated list of messages within a thread.
func (c *Client) ListMessages(orgRef, spaceRef, boardRef, threadRef string, params *ListParams) ([]Entity, *PageInfo, error) {
	return c.listEntities(buildListURL(fmt.Sprintf("/v1/orgs/%s/spaces/%s/boards/%s/threads/%s/messages", orgRef, spaceRef, boardRef, threadRef), params))
}

// GetMessage returns a single message.
func (c *Client) GetMessage(orgRef, spaceRef, boardRef, threadRef, messageRef string) (Entity, error) {
	return c.getEntity(fmt.Sprintf("/v1/orgs/%s/spaces/%s/boards/%s/threads/%s/messages/%s", orgRef, spaceRef, boardRef, threadRef, messageRef))
}

// --- Search ---

// Search performs a search across entities.
func (c *Client) Search(query string, params *ListParams) ([]Entity, *PageInfo, error) {
	if params == nil {
		params = &ListParams{}
	}
	if params.Query == nil {
		params.Query = map[string]string{}
	}
	params.Query["q"] = query
	return c.listEntities(buildListURL("/v1/search", params))
}

// --- Pipeline ---

// TransitionStage transitions a thread to a new pipeline stage.
func (c *Client) TransitionStage(orgRef, spaceRef, boardRef, threadRef, stage string) (Entity, error) {
	path := fmt.Sprintf("/v1/orgs/%s/spaces/%s/boards/%s/threads/%s/stage", orgRef, spaceRef, boardRef, threadRef)
	return c.createEntity(path, map[string]any{"stage": stage})
}

// --- Generic helpers ---

func (c *Client) listEntities(path string) ([]Entity, *PageInfo, error) {
	data, _, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	var listResp ListResponse
	if err := json.Unmarshal(data, &listResp); err != nil {
		// Try direct array.
		var entities []Entity
		if err2 := json.Unmarshal(data, &entities); err2 != nil {
			return nil, nil, fmt.Errorf("parsing list response: %w", err)
		}
		return entities, nil, nil
	}

	var entities []Entity
	if listResp.Data != nil {
		if err := json.Unmarshal(listResp.Data, &entities); err != nil {
			return nil, nil, fmt.Errorf("parsing entities: %w", err)
		}
	}
	return entities, listResp.PageInfo, nil
}

func (c *Client) getEntity(path string) (Entity, error) {
	data, _, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var entity Entity
	if err := json.Unmarshal(data, &entity); err != nil {
		return nil, fmt.Errorf("parsing entity: %w", err)
	}
	return entity, nil
}

func (c *Client) createEntity(path string, body map[string]any) (Entity, error) {
	data, _, err := c.doRequest(http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	var entity Entity
	if err := json.Unmarshal(data, &entity); err != nil {
		return nil, fmt.Errorf("parsing entity: %w", err)
	}
	return entity, nil
}

func (c *Client) updateEntity(path string, body map[string]any) (Entity, error) {
	data, _, err := c.doRequest(http.MethodPatch, path, body)
	if err != nil {
		return nil, err
	}
	var entity Entity
	if err := json.Unmarshal(data, &entity); err != nil {
		return nil, fmt.Errorf("parsing entity: %w", err)
	}
	return entity, nil
}

// ListAll fetches all pages of a paginated list. Calls listFn repeatedly until no more pages.
func ListAll[T any](listFn func(cursor string) ([]T, *PageInfo, error)) ([]T, error) {
	var all []T
	cursor := ""
	for {
		items, pageInfo, err := listFn(cursor)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if pageInfo == nil || !pageInfo.HasMore || pageInfo.NextCursor == "" {
			break
		}
		cursor = pageInfo.NextCursor
	}
	return all, nil
}

// RawGet performs a raw GET request and returns the response body as bytes.
func (c *Client) RawGet(path string) ([]byte, error) {
	data, _, err := c.doRequest(http.MethodGet, path, nil)
	return data, err
}
