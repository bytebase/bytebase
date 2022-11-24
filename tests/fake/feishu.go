package fake

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/plugin/app/feishu"
)

// Feishu is a fake feishu API server implementation for testing.
type Feishu struct {
	port  int
	echo  *echo.Echo
	botID string

	mutex sync.Mutex
	// mutex protects everything below.
	approvalDefinition map[string]bool
	approvalInstance   map[string]*approval
	// email to user id mapping.
	users map[string]string
	// user id set
	userIDs map[string]bool
}

// FeishuProviderCreator is the function to create a fake feishu provider.
type FeishuProviderCreator func(int) *Feishu

type approval struct {
	approvalCode string
	instanceCode string
	status       feishu.ApprovalStatus
}

var _ FeishuProviderCreator = NewFeishu

// NewFeishu creates a new fake feishu provider.
func NewFeishu(port int) *Feishu {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	f := &Feishu{
		port:               port,
		echo:               e,
		approvalDefinition: map[string]bool{},
		approvalInstance:   map[string]*approval{},
		users:              map[string]string{},
		userIDs:            map[string]bool{},
		mutex:              sync.Mutex{},
		botID:              uuid.NewString(),
	}

	// Routes
	g := e.Group("/open-apis")
	g.POST("/approval/v4/approvals", f.createApprovalDefinition)
	g.POST("/approval/v4/instances", f.createApprovalInstance)
	g.GET("/approval/v4/instances/:id", f.getApprovalInstanceStatus)
	g.POST("/approval/v4/instances/:id/comments", f.createApprovalInstanceComment)
	g.POST("/approval/v4/instances/cancel", f.cancelApprovalInstance)
	g.POST("/contact/v3/users/batch_get_id", f.getIDByEmail)
	g.GET("/bot/v3/info", f.getBotID)

	return f
}

// Run starts the fake feishu provider server.
func (f *Feishu) Run() error {
	return f.echo.Start(fmt.Sprintf(":%d", f.port))
}

// Close closes the fake feishu provider server.
func (f *Feishu) Close() error {
	return f.echo.Close()
}

// ListenerAddr returns the fake feishu provider server listener address.
func (f *Feishu) ListenerAddr() net.Addr {
	return f.echo.ListenerAddr()
}

// APIURL returns the API path.
func (*Feishu) APIURL(hostURL string) string {
	return fmt.Sprintf("%s/open-apis", hostURL)
}

// RegisterEmails creates a email-id pair.
func (f *Feishu) RegisterEmails(emails ...string) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	for _, email := range emails {
		_, ok := f.users[email]
		if ok {
			return errors.Errorf("register email %s twice", email)
		}
		id := uuid.NewString()
		f.users[email] = id
		f.userIDs[id] = true
	}
	return nil
}

// PendingApprovalCount returns the number of pending approvals.
func (f *Feishu) PendingApprovalCount() int {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	count := 0
	for _, approval := range f.approvalInstance {
		if approval.status == feishu.ApprovalStatusPending {
			count++
		}
	}
	return count
}

// ApprovePendingApprovals approves all pending approvals.
func (f *Feishu) ApprovePendingApprovals() {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	for _, approval := range f.approvalInstance {
		if approval.status == feishu.ApprovalStatusPending {
			approval.status = feishu.ApprovalStatusApproved
		}
	}
}

func (f *Feishu) createApprovalDefinition(c echo.Context) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	id := uuid.NewString()
	f.approvalDefinition[id] = true
	return c.JSON(http.StatusOK, &feishu.ApprovalDefinitionResponse{
		Code: 0,
		Data: struct {
			ApprovalCode string `json:"approval_code"`
		}{
			ApprovalCode: id,
		},
		Msg: "success",
	})
}

func (f *Feishu) createApprovalInstance(c echo.Context) error {
	b, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return errors.Wrap(err, "failed to read create approval instance body")
	}
	create := &feishu.CreateApprovalInstanceRequest{}
	if err := json.Unmarshal(b, create); err != nil {
		return errors.Wrap(err, "failed to unmarshal create approval instance body")
	}
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if !f.approvalDefinition[create.ApprovalCode] {
		return errors.Errorf("not found approval code %s", create.ApprovalCode)
	}
	if !f.userIDs[create.OpenID] {
		return errors.Errorf("not found user id %s", create.OpenID)
	}
	for _, idList := range create.NodeApproverOpenIDList {
		for _, id := range idList.Value {
			if !f.userIDs[id] {
				return errors.Errorf("not found user id %s", id)
			}
		}
	}

	id := uuid.NewString()
	f.approvalInstance[id] = &approval{
		approvalCode: create.ApprovalCode,
		instanceCode: id,
		status:       feishu.ApprovalStatusPending,
	}

	return c.JSON(http.StatusOK, &feishu.ExternalApprovalResponse{
		Code: 0,
		Msg:  "success",
		Data: struct {
			InstanceCode string `json:"instance_code"`
		}{
			InstanceCode: id,
		},
	})
}

func (f *Feishu) createApprovalInstanceComment(c echo.Context) error {
	userID := c.QueryParam("user_id")
	instanceID := c.Param("id")
	b, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return errors.Wrap(err, "failed to read create approval instance body")
	}
	create := &feishu.CreateExternalApprovalCommentRequest{}
	if err := json.Unmarshal(b, create); err != nil {
		return errors.Wrap(err, "failed to unmarshal create approval instance body")
	}
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if _, ok := f.approvalInstance[instanceID]; !ok {
		return errors.Errorf("not found approval instance code %s", instanceID)
	}
	if !f.userIDs[userID] {
		return errors.Errorf("not found user id %s", userID)
	}

	return c.JSON(http.StatusOK, &feishu.CreateExternalApprovalCommentResponse{
		Code: 0,
		Msg:  "",
	})
}

func (f *Feishu) getApprovalInstanceStatus(c echo.Context) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	id := c.Param("id")
	approval, ok := f.approvalInstance[id]
	if !ok {
		return errors.Errorf("not found approval instance %s", id)
	}
	return c.JSON(http.StatusOK, &feishu.GetExternalApprovalResponse{
		Code: 0,
		Msg:  "success",
		Data: struct {
			Status feishu.ApprovalStatus `json:"status"`
		}{
			Status: approval.status,
		},
	})
}

func (f *Feishu) cancelApprovalInstance(c echo.Context) error {
	b, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return errors.Wrap(err, "failed to read cancel approval instance body")
	}
	req := &feishu.CancelExternalApprovalRequest{}
	if err := json.Unmarshal(b, req); err != nil {
		return errors.Wrap(err, "failed to unmarshal cancel approval instance body")
	}

	f.mutex.Lock()
	defer f.mutex.Unlock()
	approval, ok := f.approvalInstance[req.InstanceCode]
	if !ok {
		return errors.Errorf("not found approval %s", req.InstanceCode)
	}
	if !f.userIDs[req.UserID] {
		return errors.Errorf("not found user id %s", req.UserID)
	}
	if approval.status != feishu.ApprovalStatusPending {
		return errors.Errorf(`expect to cancel a "pending" approval, but get status %q`, approval.status)
	}
	approval.status = feishu.ApprovalStatusCanceled

	return c.JSON(http.StatusOK, &feishu.CancelExternalApprovalResponse{
		Code: 0,
		Msg:  "success",
	})
}

func (f *Feishu) getIDByEmail(c echo.Context) error {
	b, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return errors.Wrap(err, "failed to read get id by email body")
	}
	req := &feishu.GetIDByEmailRequest{}
	if err := json.Unmarshal(b, req); err != nil {
		return errors.Wrap(err, "failed to unmarshal get id by email body")
	}
	f.mutex.Lock()
	defer f.mutex.Unlock()
	resp := &feishu.EmailsFindResponse{
		Code: 0,
		Msg:  "success",
	}
	for _, email := range req.Emails {
		id, ok := f.users[email]
		if !ok {
			resp.Data.UserList = append(resp.Data.UserList, struct {
				UserID string `json:"user_id"`
				Email  string `json:"email"`
			}{
				UserID: "",
				Email:  email,
			})
		} else {
			resp.Data.UserList = append(resp.Data.UserList, struct {
				UserID string `json:"user_id"`
				Email  string `json:"email"`
			}{
				UserID: id,
				Email:  email,
			})
		}
	}

	return c.JSON(http.StatusOK, resp)
}

func (f *Feishu) getBotID(c echo.Context) error {
	return c.JSON(http.StatusOK, &feishu.GetBotIDResponse{
		Code: 0,
		Msg:  "ok",
		Bot: struct {
			OpenID string `json:"open_id"`
		}{
			OpenID: f.botID,
		},
	})
}
