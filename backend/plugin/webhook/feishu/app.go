package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

// Response code definition in feishu response body.
// https://open.feishu.cn/document/ukTMukTMukTM/ugjM14COyUjL4ITN
const (
	emptyTokenRespCode   = 99991661
	invalidTokenRespCode = 99991663
)

type provider struct {
	id     string
	secret string
	c      *http.Client
	token  string
}

func newProvider(id, secret string) *provider {
	return &provider{
		id:     id,
		secret: secret,
		c:      &http.Client{},
	}
}

// tenantAccessTokenResponse is the response of GetTenantAccessToken.
type tenantAccessTokenResponse struct {
	Code   int    `json:"code"`
	Msg    string `json:"msg"`
	Token  string `json:"tenant_access_token"`
	Expire int    `json:"expire"`
}

// getIDByEmailRequest is the request of GetIDByEmail.
type getIDByEmailRequest struct {
	Emails []string `json:"emails"`
}

// emailsFindResponse is the response of GetIDByEmail.
type emailsFindResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		UserList []struct {
			UserID string `json:"user_id"`
			Email  string `json:"email"`
		} `json:"user_list"`
	} `json:"data"`
}

type sendMessageRequest struct {
	ReceiveID string `json:"receive_id"`
	MsgType   string `json:"msg_type"`
	Content   string `json:"content"`
}

type generalResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func Validate(ctx context.Context, id, secret, email string) error {
	p := newProvider(id, secret)
	if err := p.refreshToken(ctx); err != nil {
		return errors.Wrapf(err, "failed to refresh token")
	}
	_, err := p.getIDByEmail(ctx, []string{email})
	if err != nil {
		return errors.Wrapf(err, "failed to get id for user %s", email)
	}
	return nil
}

func (p *provider) refreshToken(ctx context.Context) error {
	const getTenantAccessTokenReq = `{"app_id": "%s","app_secret": "%s"}`
	const url = "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal"
	body := strings.NewReader(fmt.Sprintf(getTenantAccessTokenReq, p.id, p.secret))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return errors.Wrapf(err, "construct POST %s", url)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.c.Do(req)
	if err != nil {
		return errors.Wrapf(err, "POST %s", url)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "read body of POST %s", url)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("non-200 POST status code %d with body %q", resp.StatusCode, b)
	}

	var response tenantAccessTokenResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return errors.Wrapf(err, "unmarshal body from POST %s", url)
	}
	if response.Code != 0 {
		return errors.Errorf("failed to get tenant access token, code %d, msg %s", response.Code, response.Msg)
	}

	p.token = response.Token
	return nil
}

// getIDByEmail gets user ids by emails, returns email to userID mapping.
// https://open.feishu.cn/document/server-docs/contact-v3/user/batch_get_id
func (p *provider) getIDByEmail(ctx context.Context, emails []string) (map[string]string, error) {
	const url = "https://open.feishu.cn/open-apis/contact/v3/users/batch_get_id"
	body, err := json.Marshal(&getIDByEmailRequest{Emails: emails})
	if err != nil {
		return nil, err
	}

	b, err := p.do(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get user id by email")
	}

	var response emailsFindResponse
	if err := json.Unmarshal([]byte(b), &response); err != nil {
		return nil, err
	}

	if response.Code != 0 {
		return nil, errors.Errorf("failed to get id by email, code %d, msg %s", response.Code, response.Msg)
	}

	userID := make(map[string]string)
	for _, user := range response.Data.UserList {
		if user.UserID == "" {
			continue
		}
		userID[user.Email] = user.UserID
	}

	return userID, nil
}

// https://open.feishu.cn/document/server-docs/im-v1/message/create
func (p *provider) sendMessage(ctx context.Context, userID string, messageCard *WebhookCard) error {
	const url = "https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=open_id"
	content, err := json.Marshal(messageCard)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal message card")
	}
	data := sendMessageRequest{
		ReceiveID: userID,
		MsgType:   "interactive",
		Content:   string(content),
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal payload")
	}
	body, err := p.do(ctx, http.MethodPost, url, bytes)
	if err != nil {
		return errors.Wrapf(err, "failed to do send message request")
	}

	var resp generalResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return errors.Wrapf(err, "failed to unmarshal response")
	}

	if resp.Code != 0 {
		return errors.Errorf("failed to send message, code %d, msg %q", resp.Code, resp.Msg)
	}

	return nil
}

const maxRetries = 3

func (p *provider) do(ctx context.Context, method, url string, data []byte) ([]byte, error) {
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
			req.Header.Add("Authorization", "Bearer "+p.token)

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
				Code int `json:"code"`
			}
			if err := json.Unmarshal(b, &response); err != nil {
				return nil, false, errors.Errorf("failed to unmarshal response")
			}
			if response.Code == emptyTokenRespCode || response.Code == invalidTokenRespCode {
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
