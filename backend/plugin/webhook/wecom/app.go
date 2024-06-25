package wecom

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// Response code definition
// https://developer.work.weixin.qq.com/document/path/90313
const (
	emptyTokenRespCode   = 41001
	expiredTokenRespCode = 42001
)

type provider struct {
	c       *http.Client
	corpID  string
	agentID int
	secret  string
	token   string
}

func newProvider(corpID, agentID, secret string) (*provider, error) {
	agentIDInt, err := strconv.Atoi(agentID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert agentID %s to int", agentID)
	}
	return &provider{
		c:       &http.Client{},
		corpID:  corpID,
		agentID: agentIDInt,
		secret:  secret,
		token:   "",
	}, nil
}

type accessTokenResponse struct {
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
	AccessToken string `json:"access_token"`
	Expire      int    `json:"expires_in"`
}

func (p *provider) refreshToken(ctx context.Context) error {
	url, err := url.Parse("https://qyapi.weixin.qq.com/cgi-bin/gettoken")
	if err != nil {
		return errors.Wrapf(err, "failed to parse url")
	}
	q := url.Query()
	q.Set("corpid", p.corpID)
	q.Set("corpsecret", p.secret)
	url.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return errors.Wrapf(err, "construct GET %s", url)
	}
	resp, err := p.c.Do(req)
	if err != nil {
		return errors.Wrapf(err, "GET %s", url)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("received non-200 HTTP status code %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "read boyd of POST %s", url)
	}

	var payload accessTokenResponse
	if err := json.Unmarshal(b, &payload); err != nil {
		return errors.Wrapf(err, "failed to unmarshal")
	}
	if payload.ErrCode != 0 {
		return errors.Errorf("response errcode %d, errmsg %s", payload.ErrCode, payload.ErrMsg)
	}

	p.token = payload.AccessToken

	return nil
}

func (p *provider) getUserIDByEmail(ctx context.Context, email string) (string, error) {
	url, err := url.Parse("https://qyapi.weixin.qq.com/cgi-bin/user/get_userid_by_email")
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse url")
	}

	requestBody, err := json.Marshal(struct {
		Email     string `json:"email"`
		EmailType int    `json:"email_type"`
	}{
		Email:     email,
		EmailType: 2,
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed to marshal request body")
	}

	resp, err := p.do(ctx, http.MethodPost, url, requestBody)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get user id")
	}

	var payload struct {
		UserID string `json:"userid"`
	}
	if err := json.Unmarshal(resp, &payload); err != nil {
		return "", errors.Wrapf(err, "failed to unmarshal payload for get user id by email")
	}

	return payload.UserID, nil
}

// https://developer.work.weixin.qq.com/document/path/90236
func (p *provider) sendMessage(ctx context.Context, userIDs []string, markdown *WebhookMarkdown) error {
	url, err := url.Parse("https://qyapi.weixin.qq.com/cgi-bin/message/send")
	if err != nil {
		return errors.Wrapf(err, "failed to parse url")
	}

	requestBody, err := json.Marshal(struct {
		ToUser   string           `json:"touser"`
		MsgType  string           `json:"msgtype"`
		AgentID  int              `json:"agentid"`
		Markdown *WebhookMarkdown `json:"markdown"`
	}{
		ToUser:   strings.Join(userIDs, "|"),
		MsgType:  "markdown",
		AgentID:  p.agentID,
		Markdown: markdown,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to marshal request body")
	}

	if _, err := p.do(ctx, http.MethodPost, url, requestBody); err != nil {
		return errors.Wrapf(err, "failed to send message")
	}
	return nil
}

func (p *provider) do(ctx context.Context, method string, url *url.URL, data []byte) ([]byte, error) {
	if p.token == "" {
		if err := p.refreshToken(ctx); err != nil {
			return nil, errors.Wrapf(err, "failed to refresh token")
		}
	}
	const maxRetries = 3
	for i := 0; i < maxRetries; i++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		b, cont, err := func() ([]byte, bool, error) {
			q := url.Query()
			q.Set("access_token", p.token)
			url.RawQuery = q.Encode()

			req, err := http.NewRequestWithContext(ctx, method, url.String(), bytes.NewReader(data))
			if err != nil {
				return nil, false, errors.Wrapf(err, "failed to construct %s %s", method, url)
			}

			req.Header.Set("Content-Type", "application/json; charset=utf-8")

			resp, err := p.c.Do(req)
			if err != nil {
				return nil, false, errors.Wrapf(err, "%s %s", method, url)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return nil, false, errors.Errorf("received non-200 HTTP code %d for %s %s", resp.StatusCode, method, url)
			}

			b, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, false, errors.Wrapf(err, "failed to read body of %s %s", method, url)
			}

			var response struct {
				ErrCode int    `json:"errcode"`
				ErrMsg  string `json:"errmsg"`
			}
			if err := json.Unmarshal(b, &response); err != nil {
				return nil, false, errors.Errorf("failed to unmarshal response")
			}
			if response.ErrCode != 0 {
				return nil, false, errors.Errorf("response errcode %d, errmsg %s", response.ErrCode, response.ErrMsg)
			}
			if response.ErrCode == emptyTokenRespCode || response.ErrCode == expiredTokenRespCode {
				if err := p.refreshToken(ctx); err != nil {
					return nil, false, errors.Wrapf(err, "failed to refresh token")
				}
				return nil, true, nil
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
