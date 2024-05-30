package mediawiki

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type WikiClient struct {
	Username   string
	Password   string
	BaseURL    string
	RetryAfter time.Time
	mut        sync.Mutex
	client     *http.Client
	cookies    map[string]string
}

func NewWikiClient(username, password, baseURL string) (*WikiClient, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	return &WikiClient{
		Username: username,
		Password: password,
		BaseURL:  baseURL,
		cookies:  make(map[string]string),
		client: &http.Client{
			Jar: jar,
		},
	}, nil
}

func (w *WikiClient) Do(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	w.mut.Lock()
	defer w.mut.Unlock()
	for i := 0; i < 3; i++ {
		log.Debug().Str("url", req.URL.String()).Msg("Making request")

		if w.RetryAfter.After(time.Now()) {
			log.Warn().Str("retry-after", w.RetryAfter.String()).Msg("Rate limited, waiting")
			select {
			case <-req.Context().Done():
				return nil, fmt.Errorf("Request cancelled while waiting for rate limit")
			case <-time.After(time.Until(w.RetryAfter)):
				// Do nothing
			}
		}
		log.Info().Interface("cookies", w.cookies).Msg("Cookies")
		for k, v := range w.cookies {
			req.AddCookie(&http.Cookie{
				Name:  k,
				Value: v,
			})
		}
		resp, err = w.client.Do(req)
		if err != nil {
			log.Error().Err(err).Msg("Error making request")
			continue
		}
		defer resp.Body.Close()
		rawBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Error().Err(err).Msg("Error reading response body")
			continue
		}

		errResp := ErrorResponse{}
		err = json.Unmarshal(rawBody, &errResp)
		if err != nil {
			return nil, fmt.Errorf("Error response: %v", err)
		}
		if len(errResp.Errors) > 0 {
			for _, e := range errResp.Errors {
				if e.Code == "ratelimited" {
					w.RetryAfter = time.Now().Add(10 * time.Second)
					continue
				}
			}
		}
		resp.Body = io.NopCloser(bytes.NewBuffer(rawBody))
		for _, c := range resp.Cookies() {
			w.cookies[c.Name] = c.Value
		}
		break
	}
	return resp, err
}

func (w *WikiClient) GetLoginToken() (string, error) {
	params := map[string]string{
		"action": "query",
		"meta":   "tokens",
		"type":   "login",
		"format": "json",
	}
	req, err := http.NewRequest("GET", w.BaseURL, nil)
	if err != nil {
		return "", err
	}
	q := req.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()
	resp, err := w.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	tokenResp := TokenResponse{}
	err = json.NewDecoder(resp.Body).Decode(&tokenResp)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(tokenResp.Query.Tokens.LoginToken, "+\\"), nil
}

func (w *WikiClient) Login() error {
	tokn, err := w.GetLoginToken()
	if err != nil {
		return fmt.Errorf("Error getting login token: %w", err)
	}
	log.Info().Str("token", tokn).Msg("Got login token")
	data := map[string]string{
		"action":     "login",
		"format":     "json",
		"lgname":     w.Username,
		"lgpassword": w.Password,
		"lgtoken":    tokn,
	}
	// params := map[string]string{
	// 	"action": "login",
	// 	"format": "json",
	// }

	dataStr, _ := json.Marshal(data)

	req, err := http.NewRequest("POST", w.BaseURL, bytes.NewBuffer(dataStr))
	if err != nil {
		return err
	}
	// q := req.URL.Query()
	// for k, v := range params {
	// 	q.Add(k, v)
	// }
	// req.URL.RawQuery = q.Encode()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "SwyytchBot")

	resp, err := w.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respStr, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Debug().Str("status", resp.Status).Str("response", string(respStr)).Msg("Login response")
	return nil
}
