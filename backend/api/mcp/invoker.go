package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/bytebase/bytebase/backend/common"
)

// Invoker handles invoking gRPC methods via the internal ConnectRPC client.
type Invoker struct {
	httpClient *http.Client
	baseURL    string
}

// NewInvoker creates a new gRPC method invoker.
func NewInvoker(baseURL string) *Invoker {
	return &Invoker{
		httpClient: &http.Client{},
		baseURL:    baseURL,
	}
}

// Invoke calls a gRPC method with the given arguments.
func (i *Invoker) Invoke(
	ctx context.Context,
	svc protoreflect.ServiceDescriptor,
	method protoreflect.MethodDescriptor,
	args any,
) (proto.Message, error) {
	// Create request message dynamically
	reqType, err := protoregistry.GlobalTypes.FindMessageByName(method.Input().FullName())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find request type %s", method.Input().FullName())
	}

	reqMsg := reqType.New().Interface()

	// Unmarshal JSON args into proto message
	if args != nil {
		// Args can be json.RawMessage or map[string]any from MCP SDK
		var argsBytes []byte
		switch v := args.(type) {
		case json.RawMessage:
			argsBytes = v
		default:
			argsBytes, err = json.Marshal(v)
			if err != nil {
				return nil, errors.Wrap(err, "failed to marshal arguments")
			}
		}

		if len(argsBytes) > 0 {
			if err := common.ProtojsonUnmarshaler.Unmarshal(argsBytes, reqMsg); err != nil {
				return nil, errors.Wrap(err, "failed to unmarshal request")
			}
		}
	}

	// Build the ConnectRPC endpoint URL
	// Format: /bytebase.v1.ServiceName/MethodName
	procedure := fmt.Sprintf("/%s/%s", svc.FullName(), method.Name())
	url := i.baseURL + procedure

	// Create response message dynamically
	respType, err := protoregistry.GlobalTypes.FindMessageByName(method.Output().FullName())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find response type %s", method.Output().FullName())
	}

	respMsg := respType.New().Interface()

	// Make the ConnectRPC call
	if err := i.doConnect(ctx, url, reqMsg, respMsg); err != nil {
		return nil, err
	}

	return respMsg, nil
}

// doConnect performs a ConnectRPC unary call.
func (i *Invoker) doConnect(ctx context.Context, url string, req, resp proto.Message) error {
	// Serialize request
	reqBody, err := protojson.Marshal(req)
	if err != nil {
		return errors.Wrap(err, "failed to marshal request")
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(reqBody)))
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Forward auth header if present in context
	if auth := ctx.Value(authHeaderKey{}); auth != nil {
		httpReq.Header.Set("Authorization", auth.(string))
	}

	// Make request
	httpResp, err := i.httpClient.Do(httpReq)
	if err != nil {
		return errors.Wrap(err, "request failed")
	}
	defer httpResp.Body.Close()

	// Check for errors
	if httpResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(httpResp.Body)
		return errors.Errorf("request failed with status %d: %s", httpResp.StatusCode, string(bodyBytes))
	}

	// Deserialize response
	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read response")
	}

	if err := common.ProtojsonUnmarshaler.Unmarshal(bodyBytes, resp); err != nil {
		return errors.Wrap(err, "failed to unmarshal response")
	}

	return nil
}

// authHeaderKey is the context key for the auth header.
type authHeaderKey struct{}

// WithAuthHeader returns a context with the auth header set.
func WithAuthHeader(ctx context.Context, auth string) context.Context {
	return context.WithValue(ctx, authHeaderKey{}, auth)
}
