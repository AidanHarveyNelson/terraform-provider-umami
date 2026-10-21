// Copyright (c) Aidan Harvey-Nelson
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

const unauthorizedBody = `{"error":{"message":"Unauthorized","code":"unauthorized","status":401}}`

func TestNewClientRequiresCredentials(t *testing.T) {
	_, err := NewClient(context.Background(), ClientConfig{Endpoint: "https://umami.example.com/api"})
	if err == nil {
		t.Fatal("expected an error without credentials")
	}
}

func TestNewClientRejectsInvalidEndpoint(t *testing.T) {
	_, err := NewClient(context.Background(), ClientConfig{Endpoint: "not a url", APIKey: "umami_x"})
	if err == nil {
		t.Fatal("expected an error for an invalid endpoint")
	}
}

func TestAPIKeySentInBothHeaders(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer umami_key" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get(apiKeyHeader); got != "umami_key" {
			t.Errorf("%s = %q", apiKeyHeader, got)
		}
		writeJSON(w, 200, `{"user":{"id":"u1"}}`)
	})

	c, err := NewClient(context.Background(), ClientConfig{Endpoint: srv.URL + "/", APIKey: "umami_key"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetMe(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestLoginUsesToken(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["username"] != "admin" || body["password"] != "pw" {
				writeJSON(w, 401, unauthorizedBody)
				return
			}
			writeJSON(w, 200, `{"token":"tok"}`)
		case "/me":
			if r.Header.Get("Authorization") != "Bearer tok" {
				writeJSON(w, 401, unauthorizedBody)
				return
			}
			writeJSON(w, 200, `{"user":{"id":"u1"}}`)
		}
	})

	c, err := NewClient(context.Background(), ClientConfig{Endpoint: srv.URL, Username: "admin", Password: "pw"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetMe(context.Background()); err != nil {
		t.Fatal(err)
	}

	if _, err := NewClient(context.Background(), ClientConfig{Endpoint: srv.URL, Username: "admin", Password: "bad"}); err == nil {
		t.Fatal("expected login with a bad password to fail")
	}
}

func TestLoginRejectsTwoFactor(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, `{"requiresTwoFactor":true,"partialToken":"p"}`)
	})

	_, err := NewClient(context.Background(), ClientConfig{Endpoint: srv.URL, Username: "admin", Password: "pw"})
	if err == nil {
		t.Fatal("expected an error for a 2FA user")
	}
}

func TestGetWebsiteNotFound(t *testing.T) {
	cases := map[string]struct {
		status int
		body   string
		meOK   bool
		want   error
	}{
		"admin null body":        {status: 200, body: `null`, want: ErrNotFound},
		"soft deleted":           {status: 200, body: `{"id":"w1","name":"x","deletedAt":"2026-01-01T00:00:00Z"}`, want: ErrNotFound},
		"404":                    {status: 404, body: `{"error":{"message":"Not found"}}`, want: ErrNotFound},
		"401 valid credentials":  {status: 401, body: unauthorizedBody, meOK: true, want: ErrNotFound},
		"401 invalid credential": {status: 401, body: unauthorizedBody, meOK: false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/me" {
					if tc.meOK {
						writeJSON(w, 200, `{"user":{"id":"u1"}}`)
					} else {
						writeJSON(w, 401, unauthorizedBody)
					}
					return
				}
				writeJSON(w, tc.status, tc.body)
			})
			c, _ := NewClient(context.Background(), ClientConfig{Endpoint: srv.URL, APIKey: "umami_k"})

			_, err := c.GetWebsite(context.Background(), "w1")
			if tc.want != nil {
				if !errors.Is(err, tc.want) {
					t.Fatalf("err = %v, want %v", err, tc.want)
				}
				return
			}
			var apiErr *APIError
			if !errors.As(err, &apiErr) || apiErr.StatusCode != 401 || apiErr.Code != "unauthorized" {
				t.Fatalf("err = %v, want the 401 APIError", err)
			}
		})
	}
}

func TestCreateTeamParsesTuple(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, `[{"id":"t1","name":"team","accessCode":"team_x"},{"id":"tu1","teamId":"t1","role":"team-owner"}]`)
	})
	c, _ := NewClient(context.Background(), ClientConfig{Endpoint: srv.URL, APIKey: "umami_k"})

	team, err := c.CreateTeam(context.Background(), "team")
	if err != nil {
		t.Fatal(err)
	}
	if team.ID != "t1" || team.AccessCode == nil || *team.AccessCode != "team_x" {
		t.Fatalf("unexpected team %+v", team)
	}
}

func TestDomainRegexMatchesUmami(t *testing.T) {
	for _, d := range []string{"example.com", "ch-dev.example.com", "localhost", "localhost:3000", "a_b.example.io", "xn--bcher-kva.example"} {
		if !domainRegex.MatchString(d) {
			t.Errorf("%q should be valid", d)
		}
	}
	for _, d := range []string{"https://example.com", "example.com/path", "Example.com", "example", ""} {
		if domainRegex.MatchString(d) {
			t.Errorf("%q should be invalid", d)
		}
	}
}

func TestUUIDRegexMatchesUmami(t *testing.T) {
	if !uuidRegex.MatchString("c57d9b89-5539-4549-beae-9dc63f75e1f7") {
		t.Error("v4 UUID should be valid")
	}
	if uuidRegex.MatchString("3f8a1c2e-9b7d-1e4f-0a6b-2c3d4e5f6a7b") {
		t.Error("UUID with an invalid variant should be rejected, as Umami does")
	}
}

func TestAPIErrorIncludesFieldErrors(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 400, `{"error":{"message":"Bad request","code":"bad-request","status":400,"errors":[],"properties":{"id":{"errors":["Invalid UUID"]}}}}`)
	})
	c, _ := NewClient(context.Background(), ClientConfig{Endpoint: srv.URL, APIKey: "umami_k"})

	_, err := c.CreateWebsite(context.Background(), CreateWebsiteRequest{Name: "x", Domain: "x.com"})
	if err == nil || !strings.Contains(err.Error(), "id: Invalid UUID") {
		t.Fatalf("err = %v, want field errors", err)
	}
}

func TestAPIKeyFallsBackToLogin(t *testing.T) {
	cases := map[string]struct {
		keyValid  bool
		wantLogin bool
	}{
		"key accepted": {keyValid: true},
		"key rejected": {keyValid: false, wantLogin: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			loggedIn := false
			srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/auth/login":
					loggedIn = true
					writeJSON(w, 200, `{"token":"tok"}`)
				case "/me":
					auth := r.Header.Get("Authorization")
					if auth == "Bearer tok" || (tc.keyValid && auth == "Bearer umami_key") {
						writeJSON(w, 200, `{"user":{"id":"u1"}}`)
						return
					}
					writeJSON(w, 401, unauthorizedBody)
				}
			})

			c, err := NewClient(context.Background(), ClientConfig{Endpoint: srv.URL, APIKey: "umami_key", Username: "admin", Password: "pw"})
			if err != nil {
				t.Fatal(err)
			}
			if loggedIn != tc.wantLogin {
				t.Fatalf("logged in = %v, want %v", loggedIn, tc.wantLogin)
			}
			if _, err := c.GetMe(context.Background()); err != nil {
				t.Fatalf("GetMe after configure: %v", err)
			}
		})
	}
}

func TestAPIKeyRejectedAndLoginFails(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 401, unauthorizedBody)
	})

	_, err := NewClient(context.Background(), ClientConfig{Endpoint: srv.URL, APIKey: "umami_key", Username: "admin", Password: "bad"})
	if err == nil || !strings.Contains(err.Error(), "API key was rejected") {
		t.Fatalf("err = %v, want both failures reported", err)
	}
}

func TestGetTeamMemberNotFound(t *testing.T) {
	cases := map[string]struct {
		status int
		body   string
	}{
		"removed member": {status: 200, body: `null`},
		"missing team":   {status: 401, body: unauthorizedBody},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/me" {
					writeJSON(w, 200, `{"user":{"id":"u1"}}`)
					return
				}
				writeJSON(w, tc.status, tc.body)
			})
			c, _ := NewClient(context.Background(), ClientConfig{Endpoint: srv.URL, APIKey: "umami_k"})

			if _, err := c.GetTeamMember(context.Background(), "t1", "u2"); !errors.Is(err, ErrNotFound) {
				t.Fatalf("err = %v, want ErrNotFound", err)
			}
		})
	}
}

func TestReplayConfigRoundTrip(t *testing.T) {
	var cfg map[string]any
	if err := json.Unmarshal([]byte(`{"replayEnabled":true,"sampleRate":0.5,"maxDuration":60000,"maskLevel":"strict"}`), &cfg); err != nil {
		t.Fatal(err)
	}

	raw, diags := expandReplayConfig(context.Background(), flattenReplayConfig(cfg))
	if diags.HasError() {
		t.Fatal(diags)
	}
	want := `{"heatmapEnabled":false,"maskLevel":"strict","maxDuration":60000,"replayEnabled":true,"sampleRate":0.5}`
	if string(raw) != want {
		t.Fatalf("got %s, want %s", raw, want)
	}
}
