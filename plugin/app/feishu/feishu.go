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

// https://open.feishu.cn/document/ukTMukTMukTM/ugjM14COyUjL4ITN
const invalidTokenRespCode = 99991663

// FeishuProvider is the type of feishu.
type FeishuProvider struct {
	token  atomic.Value
	client *http.Client
}

type TokenRefresher func(ctx context.Context, client *http.Client, oldToken *string) error

// NewFeishuProvider returns a feishuProvider.
func NewFeishuProvider() *FeishuProvider {
	return &FeishuProvider{
		client: &http.Client{},
	}
}

type tokenCtx struct {
	appID     string
	appSecret string
	token     string
}

type minResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// TenantAccessTokenResponse is the response of GetTenantAccessToken.
type TenantAccessTokenResponse struct {
	Code   int    `json:"code"`
	Msg    string `json:"msg"`
	Token  string `json:"tenant_access_token"`
	Expire int    `json:"expire"`
}

// ApprovalDefinitionResponse is the response of CreateApprovalDefinition.
type ApprovalDefinitionResponse struct {
	Code int `json:"code"`
	Data struct {
		ApprovalCode string `json:"approval_code"`
		ApprovalID   string `json:"approval_id"`
	} `json:"data"`
	Msg string `json:"msg"`
}

// ExternalApprovalResponse is the response of CreateExternalApproval.
type ExternalApprovalResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		InstanceCode string `json:"instance_code"`
	} `json:"data"`
}

// GetIDByEmailReq is the request of finding user ids by emails.
type GetIDByEmailReq struct {
	Emails []string `json:"emails"`
}

// EmailsFindResponse is the response of GetIDByEmail.
type EmailsFindResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		UserList []user `json:"user_list"`
	} `json:"data"`
}

type user struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}

// GetExternalApprovalResponse is the response of GetExternalApproval.
type GetExternalApprovalResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Status string `json:"status"`
	} `json:"data"`
}

// CancelExternalApprovalResponse is the response of CancelExternalApproval.
type CancelExternalApprovalResponse struct {
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

func tokenRefresher(appID, appSecret string) TokenRefresher {
	return func(ctx context.Context, client *http.Client, oldToken *string) error {
		const url = "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal"
		body := strings.NewReader(fmt.Sprintf(getTenantAccessTokenReq, appID, appSecret))
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

		var response TenantAccessTokenResponse
		if err := json.Unmarshal(b, &response); err != nil {
			return errors.Wrapf(err, "unmarshal body from POST %s", url)
		}
		if response.Code != 0 {
			return errors.Errorf("failed to get tenant access token, code %d, msg %s", response.Code, response.Msg)
		}
		*oldToken = response.Token
		return nil
	}
}

func requester(ctx context.Context, client *http.Client, method, url string, token *string, body io.Reader) func() (*http.Response, error) {
	return func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, method, url, body)
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

func do(ctx context.Context, client *http.Client, method, url string, token *string, body io.Reader, tokenRefresher TokenRefresher) (code int, header http.Header, respBody string, err error) {
	return retry(ctx, client, token, tokenRefresher, requester(ctx, client, method, url, token, body))
}

const maxRetries = 3

func retry(ctx context.Context, client *http.Client, token *string, tokenRefresher TokenRefresher, f func() (*http.Response, error)) (code int, header http.Header, respBody string, err error) {
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

		var response minResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return 0, nil, "", errors.New("failed to unmarshal response")
		}
		if response.Code == invalidTokenRespCode {
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
// https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/reference/approval-v4/approval/create
func (p *FeishuProvider) CreateApprovalDefinition(ctx context.Context, tokenCtx tokenCtx, approvalCode string) (string, error) {
	body := strings.NewReader(fmt.Sprintf(createApprovalDefinitionReq, approvalCode))
	const url = "https://open.feishu.cn/open-apis/approval/v4/approvals"
	code, _, b, err := do(ctx, p.client, http.MethodPost, url, &tokenCtx.token, body, tokenRefresher(tokenCtx.appID, tokenCtx.appSecret))
	if err != nil {
		return "", errors.Wrapf(err, "POST %s", url)
	}

	if code != http.StatusOK {
		return "", errors.Errorf("non-200 POST status code %d with body %q", code, b)
	}

	var response ApprovalDefinitionResponse
	if err := json.Unmarshal([]byte(b), &response); err != nil {
		return "", err
	}

	if response.Code != 0 {
		return "", errors.Errorf("failed to create approval definition, code %d, msg %s", response.Code, response.Msg)
	}

	return response.Data.ApprovalCode, nil
}

// CreateExternalApproval creates an approval instance and returns instance code.
// The requester requests the approval of the approver.
// sample value of the requesterID & approverID: ou_3cda9c969f737aaa05e6915dce306cb9
// https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/reference/approval-v4/instance/create
func (p *FeishuProvider) CreateExternalApproval(ctx context.Context, tokenCtx tokenCtx, content Content, approvalCode string, requesterID string, approverID string) (string, error) {
	const url = "https://open.feishu.cn/open-apis/approval/v4/instances"
	formValue, err := formatForm(content)
	if err != nil {
		return "", err
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
	code, _, b, err := do(ctx, p.client, http.MethodPost, url, &tokenCtx.token, bytes.NewBuffer(body), tokenRefresher(tokenCtx.appID, tokenCtx.appSecret))
	if err != nil {
		return "", errors.Wrapf(err, "POST %s", url)
	}
	if code != http.StatusOK {
		return "", errors.Errorf("non-200 POST status code %d with body %q", code, b)
	}

	var response ExternalApprovalResponse
	if err := json.Unmarshal([]byte(b), &response); err != nil {
		return "", err
	}

	if response.Code != 0 {
		return "", errors.Errorf("failed to create approval instance, code %d, msg %s", response.Code, response.Msg)
	}

	return response.Data.InstanceCode, nil
}

// GetExternalApprovalStatus gets and returns the status of an approval instance.
// https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/reference/approval-v4/instance/get
func (p *FeishuProvider) GetExternalApprovalStatus(ctx context.Context, tokenCtx tokenCtx, instanceCode string) (string, error) {
	url := fmt.Sprintf("https://open.feishu.cn/open-apis/approval/v4/instances/%s", instanceCode)
	code, _, b, err := do(ctx, p.client, http.MethodGet, url, &tokenCtx.token, nil, tokenRefresher(tokenCtx.appID, tokenCtx.appSecret))
	if err != nil {
		return "", errors.Wrapf(err, "GET %s", url)
	}
	if code != http.StatusOK {
		return "", errors.Errorf("non-200 GET status code %d with body %q", code, b)
	}

	var response GetExternalApprovalResponse
	if err := json.Unmarshal([]byte(b), &response); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal response to GetExternalApprovalResponse")
	}
	if response.Code != 0 {
		return "", errors.Errorf("failed to get approval instance, code %d, msg %s", response.Code, response.Msg)
	}

	return response.Data.Status, nil
}

// CancelExternalApproval cancels an approval instance.
// https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/reference/approval-v4/instance/cancel
func (p *FeishuProvider) CancelExternalApproval(ctx context.Context, tokenCtx tokenCtx, approvalCode, instanceCode, userID string) error {
	const url = "https://open.feishu.cn/open-apis/approval/v4/instances/cancel"
	body := strings.NewReader(fmt.Sprintf(cancelExternalApprovalReq, approvalCode, instanceCode, userID))
	code, _, b, err := do(ctx, p.client, http.MethodPost, url, &tokenCtx.token, body, tokenRefresher(tokenCtx.appID, tokenCtx.appSecret))
	if err != nil {
		return errors.Wrapf(err, "POST %s", url)
	}
	if code != http.StatusOK {
		return errors.Errorf("non-200 POST status code %d with body %q", code, b)
	}

	var response CancelExternalApprovalResponse
	if err := json.Unmarshal([]byte(b), &response); err != nil {
		return errors.Wrap(err, "failed to unmarshal response to CancelExternalApprovalResponse")
	}

	if response.Code != 0 {
		return errors.Errorf("failed to create approval instance, code %d, msg %s", response.Code, response.Msg)
	}

	return nil
}

// GetIDByEmail gets user ids by emails, returns email to userID mapping.
// https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/reference/contact-v3/user/batch_get_id
// TODO(p0ny): cache email-id mapping.
func (p *FeishuProvider) GetIDByEmail(ctx context.Context, tokenCtx tokenCtx, emails []string) (map[string]string, error) {
	const url = "https://open.feishu.cn/open-apis/contact/v3/users/batch_get_id"
	body, err := json.Marshal(&GetIDByEmailReq{Emails: emails})
	if err != nil {
		return nil, err
	}

	code, _, b, err := do(ctx, p.client, http.MethodPost, url, &tokenCtx.token, bytes.NewBuffer(body), tokenRefresher(tokenCtx.appID, tokenCtx.appSecret))
	if err != nil {
		return nil, err
	}

	if code != http.StatusOK {
		return nil, errors.Errorf("non-200 POST status code %d with body %q", code, b)
	}

	var response EmailsFindResponse
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
