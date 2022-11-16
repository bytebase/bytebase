package feishu

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/common"
)

func TestProvider_CreateApprovalDefinition(t *testing.T) {
	a := require.New(t)
	p := NewProvider()
	p.client = &http.Client{
		Transport: &common.MockRoundTripper{
			MockRoundTrip: func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
{
    "code": 0,
    "data": {
        "approval_code": "6CDB63F9-7BFC-49BA-B13B-C120D8E37B4F",
        "approval_id": "7164674035308036124"
    },
    "msg": "success"
}
`)),
				}, nil
			},
		},
	}
	ctx := context.Background()
	approvalCode, err := p.CreateApprovalDefinition(ctx, TokenCtx{}, "")
	a.NoError(err)
	want := "6CDB63F9-7BFC-49BA-B13B-C120D8E37B4F"
	a.Equal(want, approvalCode)
}

func TestProvider_CreateExternalApproval(t *testing.T) {
	a := require.New(t)
	p := NewProvider()
	p.client = &http.Client{
		Transport: &common.MockRoundTripper{
			MockRoundTrip: func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
{
    "code": 0,
    "data": {
        "instance_code": "AEE54764-3873-4605-BFDF-F33BE3F0D6F7"
    },
    "msg": ""
}
`)),
				}, nil
			},
		},
	}
	ctx := context.Background()
	instanceCode, err := p.CreateExternalApproval(ctx, TokenCtx{}, Content{}, "", "", "")
	a.NoError(err)
	want := "AEE54764-3873-4605-BFDF-F33BE3F0D6F7"
	a.Equal(want, instanceCode)
}

func TestProvider_GetExternalApprovalStatus(t *testing.T) {
	a := require.New(t)
	p := NewProvider()
	p.client = &http.Client{
		Transport: &common.MockRoundTripper{
			MockRoundTrip: func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
{
  "code": 0,
  "data": {
    "approval_code": "6CDB63F9-7BFC-49BA-B13B-C120D8E37B4F",
    "approval_name": "Bytebase 工单",
    "department_id": "",
    "end_time": "0",
    "form": "[{\"id\":\"widget16681556660\",\"custom_id\":\"1\",\"name\":\"工单\",\"type\":\"input\",\"ext\":null,\"value\":\"isue12333\"}]",
    "open_id": "ou_04264ce8234cc96cca202f0bf48feeff",
    "reverted": false,
    "serial_number": "202211110001",
    "start_time": "1668155925149",
    "status": "PENDING",
    "task_list": [
      {
        "custom_node_id": "approve-here",
        "end_time": "0",
        "id": "7164675144512831489",
        "node_id": "774909ff62b8fca9120437aebd90556b",
        "node_name": "审批",
        "open_id": "ou_3fc2ae513625c18451b58a0067d11a78",
        "start_time": "1668155925310",
        "status": "PENDING",
        "type": "AND",
        "user_id": "66geca66"
      }
    ],
    "timeline": [
      {
        "create_time": "1668155925149",
        "ext": "{}",
        "node_key": "",
        "open_id": "ou_04264ce8234cc96cca202f0bf48feeff",
        "type": "START",
        "user_id": "36f3bcba"
      }
    ],
    "user_id": "36f3bcba",
    "uuid": ""
  },
  "msg": ""
}
`)),
				}, nil
			},
		},
	}
	ctx := context.Background()
	status, err := p.GetExternalApprovalStatus(ctx, TokenCtx{}, "")
	a.NoError(err)
	want := ApprovalStatusPending
	a.Equal(want, status)
}

func TestProvider_CancelExternalApproval(t *testing.T) {
	a := require.New(t)
	p := NewProvider()
	p.client = &http.Client{
		Transport: &common.MockRoundTripper{
			MockRoundTrip: func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
{
    "code": 0,
    "msg": "success",
    "data": {
    }
}
`)),
				}, nil
			},
		},
	}
	ctx := context.Background()
	err := p.CancelExternalApproval(ctx, TokenCtx{}, "", "", "")
	a.NoError(err)
}

func TestProvider_GetIDByEmail(t *testing.T) {
	t.Run("user id not found", func(t *testing.T) {
		a := require.New(t)
		p := NewProvider()
		p.client = &http.Client{
			Transport: &common.MockRoundTripper{
				MockRoundTrip: func(r *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`
{
    "code": 0,
    "msg": "success",
    "data": {
        "user_list": [
            {
                "user_id": "ou_979112345678741d29069abcdef089d4",
                "email": "zhangsan@a.com"
            },{
                "user_id": "",
                "email": "lisi@a.com"
            }
        ]
    }
}
`)),
					}, nil
				},
			},
		}
		ctx := context.Background()
		_, err := p.GetIDByEmail(ctx, TokenCtx{}, []string{"zhangsan@a.com", "lisi@a.com"})
		a.Error(err)
	})
	t.Run("success", func(t *testing.T) {
		a := require.New(t)
		p := NewProvider()
		p.client = &http.Client{
			Transport: &common.MockRoundTripper{
				MockRoundTrip: func(r *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`
{
    "code": 0,
    "msg": "success",
    "data": {
        "user_list": [
            {
                "user_id": "ou_979112345678741d29069abcdef089d4",
                "email": "zhangsan@a.com"
            },{
                "user_id": "ou_919112245678741d29069abcdef096af",
                "email": "lisi@a.com"
            }
        ]
    }
}
`)),
					}, nil
				},
			},
		}
		ctx := context.Background()
		user, err := p.GetIDByEmail(ctx, TokenCtx{}, []string{"zhangsan@a.com", "lisi@a.com"})
		a.NoError(err)
		a.Equal("ou_979112345678741d29069abcdef089d4", user["zhangsan@a.com"])
		a.Equal("ou_919112245678741d29069abcdef096af", user["lisi@a.com"])
	})
}
