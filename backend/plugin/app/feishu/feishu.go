// Package feishu implements feishu open api callers.
package feishu

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
)

// Response code definition in feishu response body.
// https://open.feishu.cn/document/ukTMukTMukTM/ugjM14COyUjL4ITN
const (
	emptyTokenRespCode   = 99991661
	invalidTokenRespCode = 99991663
)

const (
	timeout = 30 * time.Second
	// APIPath is the path of the Feishu API server.
	APIPath = "https://open.feishu.cn/open-apis"
)

// Provider is the provider for IM Feishu.
type Provider struct {
	APIPath string
	// cache token in memory.
	// use atomic.Value since it can be accessed concurrently.
	// we have initialized token so it is either an empty string or a valid but maybe expired token.
	Token  atomic.Value
	client *http.Client
}

type tokenRefresher func(ctx context.Context, client *http.Client, oldToken *string) error

// NewProvider returns a Provider.
func NewProvider(apiPath string) *Provider {
	p := Provider{
		APIPath: apiPath,
		client: &http.Client{
			Timeout: timeout,
		},
	}
	// initialize token
	p.Token.Store("")
	return &p
}

// ClearTokenCache clears cached token.
func (p *Provider) ClearTokenCache() {
	p.Token.Store("")
}

// TokenCtx is the token context to access feishu APIs.
type TokenCtx struct {
	AppID     string
	AppSecret string
}

// ApprovalStatus is the status of an external approval.
type ApprovalStatus string

const (
	// ApprovalStatusPending is the approval status for pending approvals.
	ApprovalStatusPending ApprovalStatus = "PENDING"
	// ApprovalStatusApproved is the approval status for approved approvals.
	ApprovalStatusApproved ApprovalStatus = "APPROVED"
	// ApprovalStatusRejected is the approval status for rejected approvals.
	ApprovalStatusRejected ApprovalStatus = "REJECTED"
	// ApprovalStatusCanceled is the approval status for canceled approvals.
	ApprovalStatusCanceled ApprovalStatus = "CANCELED"
	// ApprovalStatusDeleted is the approval status for deleted approvals.
	ApprovalStatusDeleted ApprovalStatus = "DELETED"
)

// tenantAccessTokenResponse is the response of GetTenantAccessToken.
type tenantAccessTokenResponse struct {
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

// EmailsFindResponse is the response of GetIDByEmail.
type EmailsFindResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		UserList []struct {
			UserID string `json:"user_id"`
			Email  string `json:"email"`
		} `json:"user_list"`
	} `json:"data"`
}

// GetExternalApprovalResponse is the response of GetExternalApproval.
type GetExternalApprovalResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Status ApprovalStatus `json:"status"`
	} `json:"data"`
}

// CreateExternalApprovalCommentResponse is the response of CreateExternalApprovalComment.
type CreateExternalApprovalCommentResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// CancelExternalApprovalResponse is the response of CancelExternalApproval.
type CancelExternalApprovalResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// GetBotIDResponse is the response of GetBotID.
type GetBotIDResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Bot  struct {
		OpenID string `json:"open_id"`
	} `json:"bot"`
}

// CreateApprovalInstanceRequest is the request of CreateApprovalInstance.
type CreateApprovalInstanceRequest struct {
	ApprovalCode           string `json:"approval_code"`
	Form                   string `json:"form"`
	NodeApproverOpenIDList []struct {
		Key   string   `json:"key"`
		Value []string `json:"value"`
	} `json:"node_approver_open_id_list"`
	OpenID string `json:"open_id"`
}

// CreateExternalApprovalCommentRequest is the request of CreateExternalApprovalComment.
type CreateExternalApprovalCommentRequest struct {
	Content string `json:"content"`
}

// CancelExternalApprovalRequest is the request of CancelExternalApproval.
type CancelExternalApprovalRequest struct {
	ApprovalCode string `json:"approval_code"`
	InstanceCode string `json:"instance_code"`
	UserID       string `json:"user_id"`
}

// GetIDByEmailRequest is the request of GetIDByEmail.
type GetIDByEmailRequest struct {
	Emails []string `json:"emails"`
}

// Content is the content of the approval.
type Content struct {
	Issue    string
	Stage    string
	Link     string
	TaskList []Task
	SQL      string
}

// Task is the content of a task.
type Task struct {
	Name      string
	Status    string
	Statement string
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
		"form_content": "[{\"id\":\"1\", \"type\": \"input\", \"name\":\"@i18n@widget1\"},{\"id\":\"2\", \"type\": \"input\", \"name\":\"@i18n@widget2\"},{\"id\":\"3\", \"type\": \"input\", \"name\":\"@i18n@widget3\"},{\"id\":\"4\", \"type\": \"textarea\", \"name\":\"@i18n@widget4\"},{\"id\":\"5\", \"type\": \"textarea\", \"name\":\"@i18n@widget5\"}]"
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
					"value": "链接"
				},
				{
					"key": "@i18n@widget3",
					"value": "阶段"
				},
				{
					"key": "@i18n@widget4",
					"value": "任务列表"
				},
				{
					"key": "@i18n@widget5",
					"value": "SQL"
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
					"value": "Link"
				},
				{
					"key": "@i18n@widget3",
					"value": "Stage"
				},
				{
					"key": "@i18n@widget4",
					"value": "Task List"
				},
				{
					"key": "@i18n@widget5",
					"value": "SQL"
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
)

func (p *Provider) tokenRefresher(tokenCtx TokenCtx) tokenRefresher {
	return func(ctx context.Context, client *http.Client, oldToken *string) error {
		url := fmt.Sprintf("%s/auth/v3/tenant_access_token/internal", p.APIPath)
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
		if err := resp.Body.Close(); err != nil {
			log.Warn("failed to close resp body", zap.Error(err))
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
	url := fmt.Sprintf("%s/approval/v4/approvals", p.APIPath)
	code, _, b, err := p.do(ctx, p.client, http.MethodPost, url, body, p.tokenRefresher(tokenCtx))
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

// CreateExternalApproval creates an external approval and returns instance code.
// The requester requests the approval of the approver.
// example approvalCode: 813718CE-F38D-45CA-A5C1-ACF4F564B526
// example requesterID & approverID: ou_3cda9c969f737aaa05e6915dce306cb9
// https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/reference/approval-v4/instance/create
func (p *Provider) CreateExternalApproval(ctx context.Context, tokenCtx TokenCtx, content Content, approvalCode string, requesterID string, approverID string) (string, error) {
	url := fmt.Sprintf("%s/approval/v4/instances", p.APIPath)
	formValue, err := formatForm(content)
	if err != nil {
		return "", errors.Wrapf(err, "failed to compose formValue, content %+v", content)
	}
	payload := CreateApprovalInstanceRequest{
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

	var response ExternalApprovalResponse
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
func (p *Provider) GetExternalApprovalStatus(ctx context.Context, tokenCtx TokenCtx, instanceCode string) (ApprovalStatus, error) {
	url := fmt.Sprintf("%s/approval/v4/instances/%s", p.APIPath, instanceCode)
	code, _, b, err := p.do(ctx, p.client, http.MethodGet, url, nil, p.tokenRefresher(tokenCtx))
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
		return "", errors.Errorf("failed to get external approval, code %d, msg %s", response.Code, response.Msg)
	}

	return response.Data.Status, nil
}

// CreateExternalApprovalComment comments an external approval.
// https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/reference/approval-v4/instance-comment/create
func (p *Provider) CreateExternalApprovalComment(ctx context.Context, tokenCtx TokenCtx, instanceCode string, userID string, msg string) error {
	url := fmt.Sprintf("%s/approval/v4/instances/%s/comments?user_id=%s", p.APIPath, instanceCode, userID)
	content, err := json.Marshal(struct {
		Text string `json:"text"`
	}{
		Text: msg,
	})
	if err != nil {
		return errors.Wrap(err, "failed to marshal content")
	}
	payload := CreateExternalApprovalCommentRequest{
		Content: string(content),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal payload %+v", payload)
	}
	code, _, b, err := p.do(ctx, p.client, http.MethodPost, url, body, p.tokenRefresher(tokenCtx))
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return errors.Errorf("non-200 POST status code %d with body %q", code, b)
	}

	var response CreateExternalApprovalCommentResponse
	if err := json.Unmarshal([]byte(b), &response); err != nil {
		return err
	}
	if response.Code != 0 {
		return errors.Errorf("failed to create external approval comment, code %d, msg %s", response.Code, response.Msg)
	}

	return nil
}

// CancelExternalApproval cancels an external approval.
// https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/reference/approval-v4/instance/cancel
func (p *Provider) CancelExternalApproval(ctx context.Context, tokenCtx TokenCtx, approvalCode, instanceCode, userID string) error {
	url := fmt.Sprintf("%s/approval/v4/instances/cancel", p.APIPath)
	req := &CancelExternalApprovalRequest{
		ApprovalCode: approvalCode,
		InstanceCode: instanceCode,
		UserID:       userID,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal body %+v", req)
	}
	code, _, b, err := p.do(ctx, p.client, http.MethodPost, url, body, p.tokenRefresher(tokenCtx))
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
		return errors.Errorf("failed to cancel external approval, code %d, msg %s", response.Code, response.Msg)
	}

	return nil
}

// GetIDByEmail gets user ids by emails, returns email to userID mapping.
// https://open.feishu.cn/document/uAjLw4CM/ukTMukTMukTM/reference/contact-v3/user/batch_get_id
// TODO(p0ny): cache email-id mapping.
func (p *Provider) GetIDByEmail(ctx context.Context, tokenCtx TokenCtx, emails []string) (map[string]string, error) {
	url := fmt.Sprintf("%s/contact/v3/users/batch_get_id", p.APIPath)
	body, err := json.Marshal(&GetIDByEmailRequest{Emails: emails})
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
			continue
		}
		userID[user.Email] = user.UserID
	}

	return userID, nil
}

// GetBotID gets the id of the bot.
// https://open.feishu.cn/document/ukTMukTMukTM/uAjMxEjLwITMx4CMyETM
func (p *Provider) GetBotID(ctx context.Context, tokenCtx TokenCtx) (string, error) {
	url := fmt.Sprintf("%s/bot/v3/info", p.APIPath)
	code, _, b, err := p.do(ctx, p.client, http.MethodGet, url, nil, p.tokenRefresher(tokenCtx))
	if err != nil {
		return "", errors.Wrapf(err, "POST %s", url)
	}
	if code != http.StatusOK {
		return "", errors.Errorf("non-200 POST status code %d with body %q", code, b)
	}

	var response GetBotIDResponse
	if err := json.Unmarshal([]byte(b), &response); err != nil {
		return "", err
	}
	if response.Code != 0 {
		return "", errors.Errorf("failed to get bot id, code %d, msg %s", response.Code, response.Msg)
	}

	return response.Bot.OpenID, nil
}

func formatForm(content Content) (string, error) {
	type form struct {
		ID    string `json:"id"`
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	var taskListValue strings.Builder
	if _, err := taskListValue.WriteString(fmt.Sprintf("Stage %q has %d task(s).\n", content.Stage, len(content.TaskList))); err != nil {
		return "", err
	}
	for i, task := range content.TaskList {
		if _, err := taskListValue.WriteString(fmt.Sprintf("%d. [%s] %s.\n", i+1, task.Status, task.Name)); err != nil {
			return "", err
		}
	}

	sql, err := formatFormSQL(content.TaskList)
	if err != nil {
		return "", err
	}

	forms := []form{
		{
			ID:    "1",
			Type:  "input",
			Value: content.Issue,
		},
		{
			ID:    "2",
			Type:  "input",
			Value: content.Link,
		},
		{
			ID:    "3",
			Type:  "input",
			Value: content.Stage,
		},
		{
			ID:    "4",
			Type:  "textarea",
			Value: taskListValue.String(),
		},
		{
			ID:    "5",
			Type:  "textarea",
			Value: sql,
		},
	}
	b, err := json.Marshal(forms)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func formatFormSQL(contentTaskList []Task) (string, error) {
	const taskSQLDisplayLimit = 5
	delimiter := strings.Repeat("=", 25)
	var sql strings.Builder

	sqlHashToTaskGroup := make(map[string]int)
	var taskGroup [][]int

	// divide tasks with the same SQLs to groups.
	for i, task := range contentTaskList {
		if task.Statement == "" {
			continue
		}
		hash := fmt.Sprintf("%x", sha1.Sum([]byte(task.Statement)))
		group, ok := sqlHashToTaskGroup[hash]
		if ok {
			taskGroup[group] = append(taskGroup[group], i)
		} else {
			taskGroup = append(taskGroup, []int{i})
			sqlHashToTaskGroup[hash] = len(taskGroup) - 1
		}
	}

	// Check if every task has the same SQL.
	// For tenant mode, it's very likely for tasks to have identical SQLs.
	// If there is one unique sql, and every task has the sql.
	if len(taskGroup) == 1 && len(taskGroup[0]) == len(contentTaskList) {
		if _, err := sql.WriteString(fmt.Sprintf("%s\nThe SQL statement of every task\n%s\n", delimiter, delimiter)); err != nil {
			return "", err
		}
		truncated := common.TruncateStringWithDescription(contentTaskList[taskGroup[0][0]].Statement)
		if _, err := sql.WriteString(fmt.Sprintf("%s\n\n", truncated)); err != nil {
			return "", err
		}
		return sql.String(), nil
	}

	count := 0
	for _, taskList := range taskGroup {
		if count >= taskSQLDisplayLimit {
			if _, err := sql.WriteString(fmt.Sprintf("%s\nDisplaying %d SQL statements, view more in Bytebase\n", delimiter, count)); err != nil {
				return "", err
			}
			break
		}

		if len(taskList) == 0 {
			continue
		}
		var tasks []string
		for _, taskIndex := range taskList {
			tasks = append(tasks, fmt.Sprintf("%d", taskIndex+1))
		}

		if _, err := sql.WriteString(fmt.Sprintf("%s\nThe SQL statement of task %s\n%s\n", delimiter, strings.Join(tasks, ","), delimiter)); err != nil {
			return "", err
		}
		truncated := common.TruncateStringWithDescription(contentTaskList[taskList[0]].Statement)
		if _, err := sql.WriteString(fmt.Sprintf("%s\n\n", truncated)); err != nil {
			return "", err
		}
		count++
	}
	return sql.String(), nil
}
