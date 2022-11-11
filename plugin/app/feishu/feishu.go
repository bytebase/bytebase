// Package feishu implements feishu open api callers.
package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/pkg/errors"
)

// Response code definition in feishu response body.
// https://open.feishu.cn/document/ukTMukTMukTM/ugjM14COyUjL4ITN
const (
	emptyTokenRespCode   = 99991661
	invalidTokenRespCode = 99991663
)

// Provider is the provider for IM Feishu.
type Provider struct {
	// cache token in memory.
	// use atomic.Value since it can be accessed concurrently.
	// we have initialized token so it is either an empty string or a valid but maybe expired token.
	Token  atomic.Value
	client *http.Client
}

type tokenRefresher func(ctx context.Context, client *http.Client, oldToken *string) error

// NewProvider returns a Provider.
func NewProvider() *Provider {
	p := Provider{
		client: &http.Client{},
	}
	// initialize token
	p.Token.Store("")
	return &p
}

// TokenCtx is the token context to access feishu APIs.
type TokenCtx struct {
	AppID     string
	AppSecret string
}

// tenantAccessTokenResponse is the response of GetTenantAccessToken.
type tenantAccessTokenResponse struct {
	Code   int    `json:"code"`
	Msg    string `json:"msg"`
	Token  string `json:"tenant_access_token"`
	Expire int    `json:"expire"`
}

// approvalDefinitionResponse is the response of CreateApprovalDefinition.
type approvalDefinitionResponse struct {
	Code int `json:"code"`
	Data struct {
		ApprovalCode string `json:"approval_code"`
		ApprovalID   string `json:"approval_id"`
	} `json:"data"`
	Msg string `json:"msg"`
}

// externalApprovalResponse is the response of CreateExternalApproval.
type externalApprovalResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		InstanceCode string `json:"instance_code"`
	} `json:"data"`
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

// getExternalApprovalResponse is the response of GetExternalApproval.
type getExternalApprovalResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Status string `json:"status"`
	} `json:"data"`
}

// cancelExternalApprovalResponse is the response of CancelExternalApproval.
type cancelExternalApprovalResponse struct {
	Code int      `json:"code"`
	Msg  string   `json:"msg"`
	Data struct{} `json:"data"`
}

type createApprovalInstanceReq struct {
	ApprovalCode           string `json:"approval_code"`
	Form                   string `json:"form"`
	NodeApproverOpenIDList []struct {
		Key   string   `json:"key"`
		Value []string `json:"value"`
	} `json:"node_approver_open_id_list"`
	OpenID string `json:"open_id"`
}

// Content is the content of the approval.
type Content struct {
	Issue       string
	Stage       string
	Description string
}

const (
	getTenantAccessTokenReq = `{
		"app_id": "%s",
		"app_secret": "%s"
}`
	createApprovalDefinitionReq = `{
  "approval_code": "%s",
	"approval_name": "@i18n@approval_name",
	"form": {
		"form_content": "[{\"id\":\"1\", \"type\": \"input\", \"name\":\"@i18n@widget1\"},{\"id\":\"2\", \"type\": \"input\", \"name\":\"@i18n@widget2\"},{\"id\":\"3\", \"type\": \"textarea\", \"name\":\"@i18n@widget3\"}]"
	},
	"i18n_resources": [
		{
			"is_default": "true",
			"locale": "zh-CN",
			"texts": [
				{
					"key": "@i18n@approval_name",
					"value": "Bytebase 工单"
				},
				{
					"key": "@i18n@node_name",
					"value": "审批"
				},
				{
					"key": "@i18n@widget1",
					"value": "工单"
				},
				{
					"key": "@i18n@widget2",
					"value": "阶段"
				},
				{
					"key": "@i18n@widget3",
					"value": "描述"
				}
			]
		},
    {
			"is_default": "false",
			"locale": "en-US",
			"texts": [
				{
					"key": "@i18n@approval_name",
					"value": "Bytebase Issue"
				},
				{
					"key": "@i18n@node_name",
					"value": "Approval"
				},
				{
					"key": "@i18n@widget1",
					"value": "Issue"
				},
				{
					"key": "@i18n@widget2",
					"value": "Stage"
				},
				{
					"key": "@i18n@widget3",
					"value": "Description"
				}
			]
    }
	],
	"node_list": [
		{
			"id": "START"
		},
		{
			"id": "approve-here",
			"name": "@i18n@node_name",
			"approver": [
				{
					"type": "Free"
				}
			]
		},
		{
			"id": "END"
		}
	],
	"viewers": [
		{
			"viewer_type": "NONE"
		}
	]
}`
	cancelExternalApprovalReq = `{
  "approval_code": "%s",
  "instance_code": "%s",
  "user_id": "%s"
}`
)

func (p *Provider) tokenRefresher(tokenCtx TokenCtx) tokenRefresher {
	return func(ctx context.Context, client *http.Client, oldToken *string) error {
		const url = "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal"
		body := strings.NewReader(fmt.Sprintf(getTenantAccessTokenReq, tokenCtx.AppID, tokenCtx.AppSecret))
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
		if err != nil {
			return errors.Wrapf(err, "construct POST %s", url)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
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
		*oldToken = response.Token

		// cache token
		p.Token.Store(response.Token)
		return nil
	}
}

func (p *Provider) do(ctx context.Context, client *http.Client, method, url string, body []byte, tokenRefresher tokenRefresher) (code int, header http.Header, respBody string, err error) {
	token := p.Token.Load().(string)
	return retry(ctx, client, &token, tokenRefresher, requester(ctx, client, method, url, &token, body))
}

// The type of body is []byte because it could be read multiple times but io.Reader can only be read once.
func requester(ctx context.Context, client *http.Client, method, url string, token *string, body []byte) func() (*http.Response, error) {
	return func() (*http.Response, error) {
		reader := bytes.NewReader(body)
		req, err := http.NewRequestWithContext(ctx, method, url, reader)
		if err != nil {
			return nil, errors.Wrapf(err, "construct %s %s", method, url)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", *token))
		resp, err := client.Do(req)
		if err != nil {
			return nil, errors.Wrapf(err, "%s %s", method, url)
		}
		return resp, nil
	}
}

const maxRetries = 3

func retry(ctx context.Context, client *http.Client, token *string, tokenRefresher tokenRefresher, f func() (*http.Response, error)) (code int, header http.Header, respBody string, err error) {
	var resp *http.Response
	var body []byte
	for retries := 0; retries < maxRetries; retries++ {
		select {
		case <-ctx.Done():
			return 0, nil, "", ctx.Err()
		default:
		}

		resp, err = f()
		if err != nil {
			return 0, nil, "", err
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return 0, nil, "", errors.Wrapf(err, "read response body with status code %d", resp.StatusCode)
		}

		var response struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			return 0, nil, "", errors.New("failed to unmarshal response")
		}
		if response.Code == emptyTokenRespCode || response.Code == invalidTokenRespCode {
			if err := tokenRefresher(ctx, client, token); err != nil {
				return 0, nil, "", err
			}
			continue
		}
		return resp.StatusCode, resp.Header, string(body), nil
	}
	return 0, nil, "", errors.Errorf("retries exceeded for token refresher with status code %d and body %q", resp.StatusCode, string(body))
}

// CreateApprovalDefinition creates an approval definition and returns approval code.
// example approvalCode: 813718CE-F38D-45CA-A5C1-ACF4F564B526
// https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/reference/approval-v4/approval/create
func (p *Provider) CreateApprovalDefinition(ctx context.Context, tokenCtx TokenCtx, approvalCode string) (string, error) {
	body := []byte(fmt.Sprintf(createApprovalDefinitionReq, approvalCode))
	const url = "https://open.feishu.cn/open-apis/approval/v4/approvals"
	code, _, b, err := p.do(ctx, p.client, http.MethodPost, url, body, p.tokenRefresher(tokenCtx))
	if err != nil {
		return "", errors.Wrapf(err, "POST %s", url)
	}

	if code != http.StatusOK {
		return "", errors.Errorf("non-200 POST status code %d with body %q", code, b)
	}

	var response approvalDefinitionResponse
	if err := json.Unmarshal([]byte(b), &response); err != nil {
		return "", err
	}

	if response.Code != 0 {
		return "", errors.Errorf("failed to create approval definition, code %d, msg %s", response.Code, response.Msg)
	}

	return response.Data.ApprovalCode, nil
}

// CreateExternalApproval creates an external approval and returns instance code.
// The requester requests the approval of the approver.
// example approvalCode: 813718CE-F38D-45CA-A5C1-ACF4F564B526
// example requesterID & approverID: ou_3cda9c969f737aaa05e6915dce306cb9
// https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/reference/approval-v4/instance/create
func (p *Provider) CreateExternalApproval(ctx context.Context, tokenCtx TokenCtx, content Content, approvalCode string, requesterID string, approverID string) (string, error) {
	const url = "https://open.feishu.cn/open-apis/approval/v4/instances"
	formValue, err := formatForm(content)
	if err != nil {
		return "", errors.Wrap(err, "failed to compose formValue")
	}
	payload := createApprovalInstanceReq{
		ApprovalCode: approvalCode,
		Form:         formValue,
		NodeApproverOpenIDList: []struct {
			Key   string   `json:"key"`
			Value []string `json:"value"`
		}{
			{
				Key:   "approve-here",
				Value: []string{approverID},
			},
		},
		OpenID: requesterID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", errors.Wrapf(err, "failed to marshal payload %+v", payload)
	}
	code, _, b, err := p.do(ctx, p.client, http.MethodPost, url, body, p.tokenRefresher(tokenCtx))
	if err != nil {
		return "", errors.Wrapf(err, "POST %s", url)
	}
	if code != http.StatusOK {
		return "", errors.Errorf("non-200 POST status code %d with body %q", code, b)
	}

	var response externalApprovalResponse
	if err := json.Unmarshal([]byte(b), &response); err != nil {
		return "", err
	}

	if response.Code != 0 {
		return "", errors.Errorf("failed to create external approval, code %d, msg %s", response.Code, response.Msg)
	}

	return response.Data.InstanceCode, nil
}

// GetExternalApprovalStatus gets and returns the status of an external approval.
// example instanceCode: 81D31358-93AF-92D6-7425-01A5D67C4E71
// https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/reference/approval-v4/instance/get
func (p *Provider) GetExternalApprovalStatus(ctx context.Context, tokenCtx TokenCtx, instanceCode string) (string, error) {
	url := fmt.Sprintf("https://open.feishu.cn/open-apis/approval/v4/instances/%s", instanceCode)
	code, _, b, err := p.do(ctx, p.client, http.MethodGet, url, nil, p.tokenRefresher(tokenCtx))
	if err != nil {
		return "", errors.Wrapf(err, "GET %s", url)
	}
	if code != http.StatusOK {
		return "", errors.Errorf("non-200 GET status code %d with body %q", code, b)
	}

	var response getExternalApprovalResponse
	if err := json.Unmarshal([]byte(b), &response); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal response to GetExternalApprovalResponse")
	}
	if response.Code != 0 {
		return "", errors.Errorf("failed to get external approval, code %d, msg %s", response.Code, response.Msg)
	}

	return response.Data.Status, nil
}

// CancelExternalApproval cancels an external approval.
// https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/reference/approval-v4/instance/cancel
func (p *Provider) CancelExternalApproval(ctx context.Context, tokenCtx TokenCtx, approvalCode, instanceCode, userID string) error {
	const url = "https://open.feishu.cn/open-apis/approval/v4/instances/cancel"
	body := []byte(fmt.Sprintf(cancelExternalApprovalReq, approvalCode, instanceCode, userID))
	code, _, b, err := p.do(ctx, p.client, http.MethodPost, url, body, p.tokenRefresher(tokenCtx))
	if err != nil {
		return errors.Wrapf(err, "POST %s", url)
	}
	if code != http.StatusOK {
		return errors.Errorf("non-200 POST status code %d with body %q", code, b)
	}

	var response cancelExternalApprovalResponse
	if err := json.Unmarshal([]byte(b), &response); err != nil {
		return errors.Wrap(err, "failed to unmarshal response to CancelExternalApprovalResponse")
	}

	if response.Code != 0 {
		return errors.Errorf("failed to create external approval, code %d, msg %s", response.Code, response.Msg)
	}

	return nil
}

// GetIDByEmail gets user ids by emails, returns email to userID mapping.
// https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/reference/contact-v3/user/batch_get_id
// TODO(p0ny): cache email-id mapping.
func (p *Provider) GetIDByEmail(ctx context.Context, tokenCtx TokenCtx, emails []string) (map[string]string, error) {
	const url = "https://open.feishu.cn/open-apis/contact/v3/users/batch_get_id"
	body, err := json.Marshal(
		struct {
			Emails []string `json:"emails"`
		}{Emails: emails})
	if err != nil {
		return nil, err
	}

	code, _, b, err := p.do(ctx, p.client, http.MethodPost, url, body, p.tokenRefresher(tokenCtx))
	if err != nil {
		return nil, err
	}

	if code != http.StatusOK {
		return nil, errors.Errorf("non-200 POST status code %d with body %q", code, b)
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
			return nil, errors.Errorf("id not found for email %s", user.Email)
		}
		userID[user.Email] = user.UserID
	}

	return userID, nil
}

func formatForm(content Content) (string, error) {
	type form []struct {
		ID    string `json:"id"`
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	forms := form{
		{
			ID:    "1",
			Type:  "input",
			Value: content.Issue,
		},
		{
			ID:    "2",
			Type:  "input",
			Value: content.Stage,
		},
		{
			ID:    "3",
			Type:  "textarea",
			Value: content.Description,
		},
	}
	b, err := json.Marshal(forms)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
