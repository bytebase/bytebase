package dingtalk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/pkg/errors"
)

type provider struct {
	id     string
	secret string
	robot  string
	c      *http.Client
	token  string
}

func newProvider(id, secret, robot string) *provider {
	return &provider{
		id:     id,
		secret: secret,
		robot:  robot,
		c:      &http.Client{},
	}
}

func Validate(ctx context.Context, id, secret, robot, phone string) error {
	p := newProvider(id, secret, robot)
	if err := p.refreshToken(ctx); err != nil {
		return errors.Wrapf(err, "failed to refresh token")
	}
	id, err := p.getIDByPhone(ctx, phone)
	if err != nil {
		return errors.Wrapf(err, "failed to get user id by phone")
	}
	if err := p.sendMessage(ctx, []string{id}, "test", "test"); err != nil {
		return errors.Wrapf(err, "failed to send test message")
	}
	return nil
}

func (p *provider) refreshToken(ctx context.Context) error {
	token, err := getTokenCached(ctx, p.c, p.id, p.secret)
	if err != nil {
		return errors.Wrapf(err, "failed to get token")
	}
	p.token = token
	return nil
}

var userIDCache = func() *lru.Cache[string, string] {
	cache, err := lru.New[string, string](5000)
	if err != nil {
		panic(err)
	}
	return cache
}()

// https://open.dingtalk.com/document/orgapp/query-users-by-phone-number
func (p *provider) getIDByPhone(ctx context.Context, phone string) (string, error) {
	if id, ok := userIDCache.Get(phone); ok {
		return id, nil
	}

	const url = "https://oapi.dingtalk.com/topapi/v2/user/getbymobile"
	b, err := p.do(ctx, http.MethodPost, url, []byte(fmt.Sprintf(`{"mobile":"%s"}`, phone)))
	if err != nil {
		return "", errors.Wrapf(err, "failed to get id by phone")
	}
	var response struct {
		Errcode int    `json:"errcode"`
		Errmsg  string `json:"errmsg"`
		Result  struct {
			UserID string `json:"userid"`
		} `json:"result"`
	}
	if err := json.Unmarshal(b, &response); err != nil {
		return "", errors.Wrapf(err, "failed to unmarshal response")
	}
	if response.Errcode == 60121 {
		userIDCache.Add(phone, "")
		return "", nil
	}
	if response.Errcode != 0 {
		return "", errors.Errorf("failed to get id by phone, errcode: %d, errmsg: %s", response.Errcode, response.Errmsg)
	}

	userIDCache.Add(phone, response.Result.UserID)
	return response.Result.UserID, nil
}

// https://open.dingtalk.com/document/orgapp/chatbots-send-one-on-one-chat-messages-in-batches
func (p *provider) sendMessage(ctx context.Context, userIDs []string, title, text string) error {
	const url = "https://api.dingtalk.com/v1.0/robot/oToMessages/batchSend"
	markdown, err := json.Marshal(struct {
		Title string `json:"title"`
		Text  string `json:"text"`
	}{
		Title: title,
		Text:  text,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to marshal markdown")
	}
	payload := struct {
		RobotCode string   `json:"robotCode"`
		UserIDs   []string `json:"userIds"`
		MsgKey    string   `json:"msgKey"`
		MsgParam  string   `json:"msgParam"`
	}{
		RobotCode: p.robot,
		UserIDs:   userIDs,
		MsgKey:    "sampleMarkdown",
		MsgParam:  string(markdown),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal payload")
	}
	_, err = p.do(ctx, http.MethodPost, url, payloadJSON)
	if err != nil {
		return errors.Wrapf(err, "failed to send text message")
	}

	return nil
}

func (p *provider) do(ctx context.Context, method, url string, data []byte) ([]byte, error) {
	const maxRetries = 3
	if p.token == "" {
		if err := p.refreshToken(ctx); err != nil {
			return nil, errors.Wrapf(err, "failed to refresh token")
		}
	}
	for i := 0; i < maxRetries; i++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		b, cont, err := func() ([]byte, bool, error) {
			req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(data))
			if err != nil {
				return nil, false, errors.Wrapf(err, "failed to construct %s %s", method, url)
			}

			req.Header.Set("Content-Type", "application/json; charset=utf-8")
			if strings.HasPrefix(url, "https://api.dingtalk.com") {
				req.Header.Add("x-acs-dingtalk-access-token", p.token)
			} else {
				url = url + "?access_token=" + p.token
			}

			resp, err := p.c.Do(req)
			if err != nil {
				return nil, false, errors.Wrapf(err, "%s %s", method, url)
			}
			defer resp.Body.Close()

			b, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, false, errors.Wrapf(err, "failed to read body of %s %s", method, url)
			}

			var response struct {
				Errcode int    `json:"errcode"`
				Errmsg  string `json:"errmsg"`

				Code    string `json:"code"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal(b, &response); err != nil {
				return nil, false, errors.Errorf("failed to unmarshal response")
			}
			if response.Errcode == 88 || response.Code == "InvalidAuthentication" {
				if err := p.refreshToken(ctx); err != nil {
					return nil, false, errors.Wrapf(err, "failed to refresh token")
				}
				return nil, true, nil
			}
			if resp.StatusCode != http.StatusOK {
				return nil, false, errors.Errorf("received non-200 HTTP code %d for %s %s, %+v", resp.StatusCode, method, url, response)
			}
			return b, false, nil
		}()
		if err != nil {
			return nil, err
		}
		if cont {
			continue
		}
		return b, nil
	}
	return nil, errors.Errorf("exceeds max retries for %s %s", method, url)
}

type tokenKey struct {
	id     string
	secret string
}

type tokenValue struct {
	token    string
	expireAt time.Time
}

var tokenCache = func() *lru.Cache[tokenKey, *tokenValue] {
	cache, err := lru.New[tokenKey, *tokenValue](3)
	if err != nil {
		panic(err)
	}
	return cache
}()

var tokenCacheLock sync.Mutex

func getTokenCached(ctx context.Context, c *http.Client, id, secret string) (string, error) {
	tokenCacheLock.Lock()
	defer tokenCacheLock.Unlock()

	key := tokenKey{
		id:     id,
		secret: secret,
	}

	token, ok := tokenCache.Get(key)
	if ok && time.Now().Before(token.expireAt.Add(-time.Minute)) {
		return token.token, nil
	}

	token, err := getToken(ctx, c, id, secret)
	if err != nil {
		return "", err
	}
	tokenCache.Add(key, token)

	return token.token, nil
}

// https://open.dingtalk.com/document/orgapp/obtain-the-access_token-of-an-internal-app
func getToken(ctx context.Context, c *http.Client, id, secret string) (*tokenValue, error) {
	payload := fmt.Sprintf(`{"appKey":"%s","appSecret":"%s"}`, id, secret)
	const url = "https://api.dingtalk.com/v1.0/oauth2/accessToken"
	body := strings.NewReader(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, errors.Wrapf(err, "construct POST %s", url)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "POST %s", url)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "read body of POST %s", url)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("non-200 POST status code %d with body %q", resp.StatusCode, b)
	}

	var response struct {
		Token  string `json:"accessToken"`
		Expire int    `json:"expireIn"`
	}
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, errors.Wrapf(err, "unmarshal body from POST %s", url)
	}

	return &tokenValue{
		token:    response.Token,
		expireAt: time.Now().Add(time.Second * time.Duration(response.Expire)),
	}, nil
}
