// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Umami Cloud reads API keys from this header; self-hosted reads the bearer token. Both are sent.
const apiKeyHeader = "x-umami-api-key"

var ErrNotFound = errors.New("not found")

type APIError struct {
	Method     string
	Path       string
	StatusCode int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	msg := e.Message
	if e.Code != "" {
		msg = fmt.Sprintf("%s (%s)", msg, e.Code)
	}
	return fmt.Sprintf("%s %s: HTTP %d: %s", e.Method, e.Path, e.StatusCode, msg)
}

type ClientConfig struct {
	Endpoint string
	APIKey   string
	Username string
	Password string

	UserAgent  string
	HTTPClient *http.Client
}

type Client struct {
	endpoint   string
	apiKey     string
	token      string
	userAgent  string
	httpClient *http.Client
}

func NewClient(ctx context.Context, cfg ClientConfig) (*Client, error) {
	endpoint := strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
	if endpoint == "" {
		return nil, errors.New("endpoint is required")
	}
	if _, err := url.ParseRequestURI(endpoint); err != nil {
		return nil, fmt.Errorf("invalid endpoint %q: %w", cfg.Endpoint, err)
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	c := &Client{
		endpoint:   endpoint,
		apiKey:     cfg.APIKey,
		userAgent:  cfg.UserAgent,
		httpClient: httpClient,
	}

	hasLogin := cfg.Username != "" && cfg.Password != ""

	switch {
	case cfg.APIKey != "" && hasLogin:
		// Umami before v3.4.0 rejects API keys, so fall back to logging in.
		_, err := c.GetMe(ctx)
		if err == nil {
			return c, nil
		}
		if !isUnauthorized(err) {
			return nil, err
		}
		c.apiKey = ""
		if err := c.login(ctx, cfg.Username, cfg.Password); err != nil {
			return nil, fmt.Errorf("the API key was rejected and %w", err)
		}
		return c, nil
	case cfg.APIKey != "":
		return c, nil
	case hasLogin:
		if err := c.login(ctx, cfg.Username, cfg.Password); err != nil {
			return nil, err
		}
		return c, nil
	default:
		return nil, errors.New("either an API key or a username and password must be configured")
	}
}

func isUnauthorized(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && (apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden)
}

type loginResponse struct {
	Token             string `json:"token"`
	RequiresTwoFactor bool   `json:"requiresTwoFactor"`
}

func (c *Client) login(ctx context.Context, username, password string) error {
	var res loginResponse
	body := map[string]string{"username": username, "password": password}
	if err := c.do(ctx, http.MethodPost, "/auth/login", body, &res); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	if res.RequiresTwoFactor {
		return errors.New("login failed: the user has two-factor authentication enabled, which the provider cannot complete; use an API key or a user without 2FA")
	}
	if res.Token == "" {
		return errors.New("login failed: response did not contain a token")
	}
	c.token = res.Token
	return nil
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	switch {
	case c.apiKey != "":
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set(apiKeyHeader, c.apiKey)
	case c.token != "":
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return newAPIError(method, path, res, data)
	}

	if out == nil || len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("%s %s: decoding response: %w", method, path, err)
	}
	return nil
}

func newAPIError(method, path string, res *http.Response, data []byte) *APIError {
	apiErr := &APIError{Method: method, Path: path, StatusCode: res.StatusCode}

	var envelope struct {
		Error json.RawMessage `json:"error"`
	}
	if json.Unmarshal(data, &envelope) == nil && len(envelope.Error) > 0 {
		var detail struct {
			Message    string `json:"message"`
			Code       string `json:"code"`
			Properties map[string]struct {
				Errors []string `json:"errors"`
			} `json:"properties"`
		}
		if json.Unmarshal(envelope.Error, &detail) == nil {
			apiErr.Message, apiErr.Code = detail.Message, detail.Code
			fields := make([]string, 0, len(detail.Properties))
			for field, p := range detail.Properties {
				fields = append(fields, field+": "+strings.Join(p.Errors, ", "))
			}
			sort.Strings(fields)
			if len(fields) > 0 {
				apiErr.Message += " [" + strings.Join(fields, "; ") + "]"
			}
		} else {
			_ = json.Unmarshal(envelope.Error, &apiErr.Message)
		}
	}
	if apiErr.Message == "" {
		apiErr.Message = strings.TrimSpace(string(data))
	}
	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(res.StatusCode)
	}
	return apiErr
}

// Umami answers 401 rather than 404 when a non-admin requests a missing object,
// so a 401/403 means "not found" as long as the credentials still work.
func (c *Client) resolveMissing(ctx context.Context, err error) error {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return err
	}
	switch apiErr.StatusCode {
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusUnauthorized, http.StatusForbidden:
		if _, meErr := c.GetMe(ctx); meErr != nil {
			return err
		}
		return ErrNotFound
	}
	return err
}

type Me struct {
	User struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Role     string `json:"role"`
		IsAdmin  bool   `json:"isAdmin"`
	} `json:"user"`
}

func (c *Client) GetMe(ctx context.Context) (*Me, error) {
	var me Me
	if err := c.do(ctx, http.MethodGet, "/me", nil, &me); err != nil {
		return nil, err
	}
	if me.User.ID == "" {
		return nil, errors.New("GET /me: response did not contain a user id")
	}
	return &me, nil
}

type Config struct {
	CloudMode              bool    `json:"cloudMode"`
	PrivateMode            bool    `json:"privateMode"`
	SessionDeletionEnabled bool    `json:"sessionDeletionEnabled"`
	TelemetryDisabled      bool    `json:"telemetryDisabled"`
	UpdatesDisabled        bool    `json:"updatesDisabled"`
	TrackerScriptName      *string `json:"trackerScriptName"`
	FaviconURL             *string `json:"faviconUrl"`
	LinksURL               *string `json:"linksUrl"`
	PixelsURL              *string `json:"pixelsUrl"`
}

func (c *Client) GetConfig(ctx context.Context) (*Config, error) {
	var cfg Config
	if err := c.do(ctx, http.MethodGet, "/config", nil, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

type Website struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Domain    *string `json:"domain"`
	UserID    *string `json:"userId"`
	TeamID    *string `json:"teamId"`
	CreatedBy *string `json:"createdBy"`
	CreatedAt *string `json:"createdAt"`
	DeletedAt *string `json:"deletedAt"`

	ReplayConfig map[string]any `json:"replayConfig"`
}

type CreateWebsiteRequest struct {
	ID     string  `json:"id,omitempty"`
	Name   string  `json:"name"`
	Domain string  `json:"domain"`
	TeamID *string `json:"teamId,omitempty"`
}

func (c *Client) CreateWebsite(ctx context.Context, in CreateWebsiteRequest) (*Website, error) {
	var website Website
	if err := c.do(ctx, http.MethodPost, "/websites", in, &website); err != nil {
		return nil, err
	}
	if website.ID == "" {
		return nil, errors.New("POST /websites: response did not contain a website id")
	}
	return &website, nil
}

func (c *Client) GetWebsite(ctx context.Context, id string) (*Website, error) {
	var website *Website
	if err := c.do(ctx, http.MethodGet, "/websites/"+url.PathEscape(id), nil, &website); err != nil {
		return nil, c.resolveMissing(ctx, err)
	}
	// Admins get 200 with a null body for unknown ids; Umami Cloud soft-deletes.
	if website == nil || website.ID == "" || website.DeletedAt != nil {
		return nil, ErrNotFound
	}
	return website, nil
}

type UpdateWebsiteRequest struct {
	Name   string `json:"name,omitempty"`
	Domain string `json:"domain,omitempty"`

	// Omitted leaves the replay config alone, "null" resets it, and an object is merged into it.
	ReplayConfig json.RawMessage `json:"replayConfig,omitempty"`
}

func (c *Client) UpdateWebsite(ctx context.Context, id string, body UpdateWebsiteRequest) (*Website, error) {
	var website Website
	if err := c.do(ctx, http.MethodPost, "/websites/"+url.PathEscape(id), body, &website); err != nil {
		return nil, err
	}
	return &website, nil
}

func (c *Client) TransferWebsite(ctx context.Context, id, teamID, userID string) error {
	body := map[string]string{"userId": userID}
	if teamID != "" {
		body = map[string]string{"teamId": teamID}
	}
	return c.do(ctx, http.MethodPost, "/websites/"+url.PathEscape(id)+"/transfer", body, nil)
}

func (c *Client) DeleteWebsite(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/websites/"+url.PathEscape(id), nil, nil)
}

type Team struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	AccessCode *string `json:"accessCode"`
	CreatedAt  *string `json:"createdAt"`
	DeletedAt  *string `json:"deletedAt"`
}

func (c *Client) CreateTeam(ctx context.Context, name string) (*Team, error) {
	// POST /teams returns [team, teamUser].
	var raw json.RawMessage
	if err := c.do(ctx, http.MethodPost, "/teams", map[string]string{"name": name}, &raw); err != nil {
		return nil, err
	}

	var team Team
	var tuple []json.RawMessage
	if err := json.Unmarshal(raw, &tuple); err == nil {
		if len(tuple) == 0 {
			return nil, errors.New("POST /teams: empty response")
		}
		raw = tuple[0]
	}
	if err := json.Unmarshal(raw, &team); err != nil {
		return nil, fmt.Errorf("POST /teams: decoding response: %w", err)
	}
	if team.ID == "" {
		return nil, errors.New("POST /teams: response did not contain a team id")
	}
	return &team, nil
}

func (c *Client) GetTeam(ctx context.Context, id string) (*Team, error) {
	var team *Team
	if err := c.do(ctx, http.MethodGet, "/teams/"+url.PathEscape(id), nil, &team); err != nil {
		return nil, c.resolveMissing(ctx, err)
	}
	if team == nil || team.ID == "" || team.DeletedAt != nil {
		return nil, ErrNotFound
	}
	return team, nil
}

func (c *Client) UpdateTeam(ctx context.Context, id, name string) (*Team, error) {
	var team Team
	if err := c.do(ctx, http.MethodPost, "/teams/"+url.PathEscape(id), map[string]string{"name": name}, &team); err != nil {
		return nil, err
	}
	return &team, nil
}

func listAll[T any](ctx context.Context, c *Client, path string) ([]T, error) {
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	var all []T
	for page := 1; ; page++ {
		var res struct {
			Data  []T `json:"data"`
			Count int `json:"count"`
		}
		if err := c.do(ctx, http.MethodGet, fmt.Sprintf("%s%spage=%d&pageSize=100", path, sep, page), nil, &res); err != nil {
			return nil, err
		}
		all = append(all, res.Data...)
		if len(res.Data) == 0 || len(all) >= res.Count {
			return all, nil
		}
	}
}

func (c *Client) ListTeamWebsites(ctx context.Context, teamID string) ([]Website, error) {
	return listAll[Website](ctx, c, "/teams/"+url.PathEscape(teamID)+"/websites")
}

// ListWebsites returns the user's websites and those of teams where the user is owner or manager.
func (c *Client) ListWebsites(ctx context.Context, search string) ([]Website, error) {
	query := url.Values{"includeTeams": {"true"}}
	if search != "" {
		query.Set("search", search)
	}
	return listAll[Website](ctx, c, "/websites?"+query.Encode())
}

func (c *Client) ListTeams(ctx context.Context) ([]Team, error) {
	return listAll[Team](ctx, c, "/teams")
}

func (c *Client) ListTeamMembers(ctx context.Context, teamID string) ([]TeamMember, error) {
	members, err := listAll[TeamMember](ctx, c, "/teams/"+url.PathEscape(teamID)+"/users")
	if err != nil {
		return nil, c.resolveMissing(ctx, err)
	}
	return members, nil
}

func (c *Client) DeleteTeam(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/teams/"+url.PathEscape(id), nil, nil)
}

type TeamMember struct {
	ID        string  `json:"id"`
	TeamID    string  `json:"teamId"`
	UserID    string  `json:"userId"`
	Role      string  `json:"role"`
	CreatedAt *string `json:"createdAt"`

	User *struct {
		Username string `json:"username"`
	} `json:"user"`
}

func teamMemberPath(teamID, userID string) string {
	return "/teams/" + url.PathEscape(teamID) + "/users/" + url.PathEscape(userID)
}

func (c *Client) AddTeamMember(ctx context.Context, teamID, userID, role string) (*TeamMember, error) {
	var member TeamMember
	body := map[string]string{"userId": userID, "role": role}
	if err := c.do(ctx, http.MethodPost, "/teams/"+url.PathEscape(teamID)+"/users", body, &member); err != nil {
		return nil, err
	}
	return &member, nil
}

func (c *Client) GetTeamMember(ctx context.Context, teamID, userID string) (*TeamMember, error) {
	var member *TeamMember
	if err := c.do(ctx, http.MethodGet, teamMemberPath(teamID, userID), nil, &member); err != nil {
		return nil, c.resolveMissing(ctx, err)
	}
	if member == nil || member.UserID == "" {
		return nil, ErrNotFound
	}
	return member, nil
}

func (c *Client) UpdateTeamMember(ctx context.Context, teamID, userID, role string) (*TeamMember, error) {
	var member TeamMember
	if err := c.do(ctx, http.MethodPost, teamMemberPath(teamID, userID), map[string]string{"role": role}, &member); err != nil {
		return nil, err
	}
	return &member, nil
}

func (c *Client) RemoveTeamMember(ctx context.Context, teamID, userID string) error {
	return c.do(ctx, http.MethodDelete, teamMemberPath(teamID, userID), nil, nil)
}

const shareTypeWebsite = 1

type Share struct {
	ID         string         `json:"id"`
	EntityID   string         `json:"entityId"`
	Name       string         `json:"name"`
	Slug       string         `json:"slug"`
	Parameters map[string]any `json:"parameters"`
	CreatedAt  *string        `json:"createdAt"`
}

type ShareRequest struct {
	EntityID   string         `json:"entityId,omitempty"`
	ShareType  int            `json:"shareType,omitempty"`
	Name       string         `json:"name"`
	Slug       string         `json:"slug,omitempty"`
	Parameters map[string]any `json:"parameters"`
}

// Umami answers a duplicate slug with an empty 500.
func shareSlugHint(err error, slug string) error {
	var apiErr *APIError
	if slug != "" && errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusInternalServerError {
		return fmt.Errorf("%w (the slug %q may already be in use)", err, slug)
	}
	return err
}

func (c *Client) CreateShare(ctx context.Context, in ShareRequest) (*Share, error) {
	var share Share
	if err := c.do(ctx, http.MethodPost, "/share", in, &share); err != nil {
		return nil, shareSlugHint(err, in.Slug)
	}
	if share.ID == "" {
		return nil, errors.New("POST /share: response did not contain a share id")
	}
	return &share, nil
}

// GET /share/id/{id} fails with a 500 for unknown ids, so shares are read from the website's list.
func (c *Client) GetWebsiteShare(ctx context.Context, websiteID, shareID string) (*Share, error) {
	for page := 1; ; page++ {
		var res struct {
			Data  []Share `json:"data"`
			Count int     `json:"count"`
		}
		path := fmt.Sprintf("/websites/%s/shares?page=%d&pageSize=100", url.PathEscape(websiteID), page)
		if err := c.do(ctx, http.MethodGet, path, nil, &res); err != nil {
			return nil, c.resolveMissing(ctx, err)
		}
		for i := range res.Data {
			if res.Data[i].ID == shareID {
				return &res.Data[i], nil
			}
		}
		if len(res.Data) == 0 || page*100 >= res.Count {
			return nil, ErrNotFound
		}
	}
}

func (c *Client) UpdateShare(ctx context.Context, id string, in ShareRequest) (*Share, error) {
	var share Share
	if err := c.do(ctx, http.MethodPost, "/share/id/"+url.PathEscape(id), in, &share); err != nil {
		return nil, shareSlugHint(err, in.Slug)
	}
	return &share, nil
}

func (c *Client) DeleteShare(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/share/id/"+url.PathEscape(id), nil, nil)
}
