package flightradar

import (
	"bufio"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Authentication struct {
    Message  string         `json:"message,omitempty"`
    User     map[string]any `json:"user,omitempty"`
    UserData map[string]any `json:"userData,omitempty"`
}

// LoginFromEnvOrConfig reads env vars and optional INI config and updates the client.
// Env:
//
//	fr24_username, fr24_password
//	fr24_subscription_key, fr24_token
//
// Config file: $XDG_CONFIG_HOME/fr24/fr24.conf with section [global] and same keys.
func (c *Client) LoginFromEnvOrConfig() error {
    creds := readCredentials()
    if creds.username != "" && creds.password != "" {
        auth, err := loginWithUsernamePassword(c.http, creds.username, creds.password)
        if err != nil {
            return err
        }
        // Extract subscriptionKey and accessToken if present
        if ud, ok := auth.UserData["subscriptionKey"].(string); ok && ud != "" {
            c.subscriptionKey = ud
        }
        if at, ok := auth.UserData["accessToken"].(string); ok && at != "" {
            c.authToken = at
        }
        return nil
    }
    if creds.subscriptionKey != "" {
        c.subscriptionKey = creds.subscriptionKey
        // token optional
        if creds.token != "" {
            c.authToken = creds.token
        }
        return nil
    }
    return nil
}

// AuthMode returns a simple string describing the current auth configuration.
// - "bearer": client has an auth token (either via login or provided token)
// - "subscription-key": client has a subscription key (JSON endpoints)
// - "anonymous": neither token nor key configured
func (c *Client) AuthMode() string {
    if c.authToken != "" {
        return "bearer"
    }
    if c.subscriptionKey != "" {
        return "subscription-key"
    }
    return "anonymous"
}

type credentials struct{ username, password, subscriptionKey, token string }

func readCredentials() credentials {
	c := credentials{
		username:        os.Getenv("fr24_username"),
		password:        os.Getenv("fr24_password"),
		subscriptionKey: os.Getenv("fr24_subscription_key"),
		token:           os.Getenv("fr24_token"),
	}
	// optional INI file override
	if dir, err := os.UserConfigDir(); err == nil {
		fp := filepath.Join(dir, "fr24", "fr24.conf")
        if f, err := os.Open(fp); err == nil {
            defer func() { _ = f.Close() }()
			// very small INI reader for [global] key=value
			s := bufio.NewScanner(f)
			inGlobal := false
			for s.Scan() {
				ln := strings.TrimSpace(s.Text())
				if ln == "" || strings.HasPrefix(ln, ";") || strings.HasPrefix(ln, "#") {
					continue
				}
				if strings.HasPrefix(ln, "[") {
					inGlobal = strings.EqualFold(ln, "[global]")
					continue
				}
				if !inGlobal {
					continue
				}
				if i := strings.Index(ln, "="); i > 0 {
					k := strings.TrimSpace(ln[:i])
					v := strings.TrimSpace(ln[i+1:])
					switch k {
					case "username":
						c.username = v
					case "password":
						c.password = v
					case "subscription_key":
						c.subscriptionKey = v
					case "token":
						c.token = v
					}
				}
			}
		}
	}
	return c
}

func loginWithUsernamePassword(httpc *http.Client, username, password string) (Authentication, error) {
	req, _ := http.NewRequest("POST", "https://www.flightradar24.com/user/login", strings.NewReader("email="+urlEncode(username)+"&password="+urlEncode(password)))
	for k, vs := range DEFAULT_JSON_HEADERS_NOAUTH() {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpc.Do(req)
	if err != nil {
		return Authentication{}, err
	}
    defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Authentication{}, errors.New("login failed: status " + resp.Status)
	}
	var auth Authentication
	if err := json.NewDecoder(resp.Body).Decode(&auth); err != nil {
		return Authentication{}, err
	}
	return auth, nil
}

func DEFAULT_JSON_HEADERS_NOAUTH() http.Header { return http.Header(defaultJSONHeaders("")) }

func urlEncode(s string) string {
	r := strings.NewReplacer("%", "%25", " ", "+", "&", "%26", "=", "%3D")
	return r.Replace(s)
}
