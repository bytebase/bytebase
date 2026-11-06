package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
)

// instanceNotActivated is a change line, never description text, which Claude
// Code's fullscreen renderer cuts to one line.
const instanceNotActivated = "Not activated, so masking skips it."

// instanceRequestFields are the CreateInstanceRequest fields the instance
// entry takes. A request that sets any other field is refused before the form,
// since the form could not state it.
var instanceRequestFields = map[string]bool{
	"parent":                     true,
	"instance_id":                true,
	"instance":                   true,
	"instance.title":             true,
	"instance.engine":            true,
	"instance.environment":       true,
	"instance.activation":        true,
	"instance.data_sources":      true,
	"instance.data_sources.id":   true,
	"instance.data_sources.type": true,
	"instance.data_sources.authentication_type":    true,
	"instance.data_sources.host":                   true,
	"instance.data_sources.port":                   true,
	"instance.data_sources.username":               true,
	"instance.data_sources.password":               true,
	"instance.data_sources.database":               true,
	"instance.data_sources.use_ssl":                true,
	"instance.data_sources.verify_tls_certificate": true,
}

// instanceCreation is a request to add a PostgreSQL instance to a project.
type instanceCreation struct {
	request   *v1pb.CreateInstanceRequest
	projectID string
}

func (*instanceCreation) confirmedProcedure() string {
	return v1connect.InstanceServiceCreateInstanceProcedure
}

// confirmedResource is the new instance's name, as the gate and the audit log
// name the instance a create adds.
func (c *instanceCreation) confirmedResource() string {
	return common.FormatProjectInstance(c.projectID, c.request.GetInstanceId())
}

func (c *instanceCreation) dataSource() *v1pb.DataSource {
	return c.request.GetInstance().GetDataSources()[0]
}

// environmentID is the instance's environment ID, or "" for none.
func (c *instanceCreation) environmentID() string {
	id, err := common.GetEnvironmentID(c.request.GetInstance().GetEnvironment())
	if err != nil {
		return ""
	}
	return id
}

// instanceSubject is an instance creation in confirmAdminChange's words.
type instanceSubject struct{}

func (instanceSubject) unchanged() string { return "no instance was added" }

func (instanceSubject) notChanged() string { return "no instance was added" }

func (instanceSubject) revised() string { return "the revised request" }

func (instanceSubject) reread() string {
	return "call administer_workspace again to ask the user about the request as it is now"
}

func (s *Server) administerInstance(ctx context.Context, req *mcp.CallToolRequest, input AdministerWorkspaceInput) (*mcp.CallToolResult, any, error) {
	change, err := parseInstanceCreation(input.Body, getWorkspaceID(ctx))
	if err != nil {
		return formatToolError(err), nil, nil
	}
	link := s.consoleLink(ctx, common.FormatProject(change.projectID)+"/instances/new")
	if err := change.check(link); err != nil {
		return formatToolError(err), nil, nil
	}
	if !canShowConfirmation(req) {
		return formatToolError(&toolError{
			Code:       "CONFIRMATION_UNAVAILABLE",
			Message:    "this MCP client cannot show the confirmation form administer_workspace needs, so no instance was added",
			Suggestion: "add it signed in to the Bytebase console: " + link,
		}), nil, nil
	}

	// Read on both rounds: an answer is acted on only while the environment the
	// form states exists.
	if err := s.checkEnvironment(ctx, change); err != nil {
		return formatToolError(err), nil, nil
	}
	maskingSkips := s.maskingSkipsUnactivated(ctx, getWorkspaceID(ctx))
	form, err := instanceForm(ctx, change, maskingSkips, link)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	digest := adminFormDigest(form, change.request)
	if result := s.confirmAdminChange(ctx, req, change, digest, form, instanceSubject{}, "this request and its form"); result != nil {
		return result, nil, nil
	}

	confirmedCtx := auth.WithMCPConfirmedDecision(ctx, auth.MCPConfirmedDecision{
		Procedure: change.confirmedProcedure(),
		Resource:  change.confirmedResource(),
	})
	// The create succeeds even when Bytebase cannot connect, and MCP can then
	// neither repair the instance nor remove it, so the connection is tested
	// first, as the console's Create does.
	test := proto.CloneOf(change.request)
	test.ValidateOnly = true
	resp, err := s.sendInstanceCreation(confirmedCtx, test)
	if why := refusal(resp, err); why != "" {
		// For the requests this entry takes, a 401 or 403 refuses the caller
		// before the test runs, so it is not a test result.
		if err == nil && (resp.Status == http.StatusUnauthorized || resp.Status == http.StatusForbidden) {
			return formatToolError(errors.Errorf("failed to add the instance: %s", why)), nil, nil
		}
		return formatToolError(&toolError{
			Code:       "CONNECTION_TEST_FAILED",
			Message:    "Bytebase's connection test refused the request, so no instance was added: " + why,
			Suggestion: "correct the request with the user as the message says, then call administer_workspace again; the user confirms the corrected request in a new form",
		}), nil, nil
	}
	resp, err = s.sendInstanceCreation(confirmedCtx, change.request)
	instanceLink := s.consoleLink(ctx, change.confirmedResource())
	created, err := s.createdInstance(ctx, change, resp, err, instanceLink)
	if err != nil {
		return formatToolError(err), nil, nil
	}
	text := s.instanceAddedText(ctx, change, created, maskingSkips)
	return textResult(text), &AdministerWorkspaceOutput{OperationID: input.OperationID, Changed: true, Message: text, Link: instanceLink}, nil
}

func (s *Server) sendInstanceCreation(ctx context.Context, request *v1pb.CreateInstanceRequest) (*apiResponse, error) {
	body, err := protojson.Marshal(request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build the request")
	}
	return s.apiRequest(ctx, v1connect.InstanceServiceCreateInstanceProcedure, json.RawMessage(body))
}

// createdInstance is the instance the create added. The create stores the
// instance before it syncs it, so a call that times out or fails with a 5xx
// may still have added it, and the instance is read back. Its data source's ID
// is a new UUID per request, so a match shows this request added it.
func (s *Server) createdInstance(ctx context.Context, c *instanceCreation, resp *apiResponse, sendErr error, link string) (*v1pb.Instance, error) {
	why := refusal(resp, sendErr)
	if why == "" {
		created := &v1pb.Instance{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(resp.Body, created); err != nil {
			return nil, errors.Wrap(err, "the instance was added, but the response could not be read")
		}
		return created, nil
	}
	if sendErr == nil && resp.Status < http.StatusInternalServerError {
		return nil, errors.Errorf("failed to add the instance: %s", why)
	}
	read, err := s.apiRequest(ctx, v1connect.InstanceServiceGetInstanceProcedure, map[string]string{"name": c.confirmedResource()})
	switch {
	case err == nil && read.Status == http.StatusOK:
		found := &v1pb.Instance{}
		if common.ProtojsonUnmarshaler.Unmarshal(read.Body, found) != nil {
			break
		}
		if addedByThisRequest(found, c) {
			return found, nil
		}
		return nil, errors.Errorf("failed to add the instance: instance ID %q already belongs to another instance, so no instance was added", c.request.GetInstanceId())
	// After a 5xx the create has returned, so no part of it is still running;
	// after a timeout it may be.
	case err == nil && read.Status == http.StatusNotFound && sendErr == nil:
		return nil, errors.Errorf("failed to add the instance: %s", why)
	}
	return nil, &toolError{
		Code:       "ADD_UNCONFIRMED",
		Message:    fmt.Sprintf("the create did not finish (%s), and Bytebase could not confirm whether instance %q was added", why, c.request.GetInstanceId()),
		Suggestion: "check the instance page before trying again, since adding the same server under another ID makes a second instance: " + link,
	}
}

func addedByThisRequest(found *v1pb.Instance, c *instanceCreation) bool {
	if found.GetState() != v1pb.State_ACTIVE {
		return false
	}
	for _, ds := range found.GetDataSources() {
		if ds.GetType() == v1pb.DataSourceType_ADMIN {
			return ds.GetId() == c.dataSource().GetId()
		}
	}
	return false
}

// refusal says why a call failed, as the handler or the transport states it,
// or "" when it succeeded.
func refusal(resp *apiResponse, err error) string {
	switch {
	case err != nil:
		return err.Error()
	case resp.Status >= http.StatusBadRequest:
		return fmt.Sprintf("HTTP %d: %s", resp.Status, parseError(resp.Body))
	default:
		return ""
	}
}

// parseInstanceCreation reads the request and the project it adds the instance
// to.
func parseInstanceCreation(body map[string]any, workspace string) (*instanceCreation, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, &toolError{Code: "INVALID_REQUEST", Message: "the request body could not be read: " + err.Error()}
	}
	request := &v1pb.CreateInstanceRequest{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(encoded, request); err != nil {
		return nil, &toolError{Code: "INVALID_REQUEST", Message: "the request body is not a CreateInstanceRequest: " + err.Error()}
	}
	parent := request.GetParent()
	projectID, err := common.GetProjectID(parent)
	if err != nil || projectID == "-" {
		return nil, &toolError{
			Code:       "UNSUPPORTED_CHANGE",
			Message:    fmt.Sprintf("administer_workspace adds an instance to a project, and the request's parent is %q", parent),
			Suggestion: `set parent to the project's name, such as "projects/hr"; add a workspace instance signed in to the Bytebase console`,
		}
	}
	if common.IsDefaultProject(workspace, projectID) {
		return nil, &toolError{
			Code:       "INVALID_REQUEST",
			Message:    fmt.Sprintf("the default project %q cannot own an instance", parent),
			Suggestion: "add the instance to the project its databases belong in",
		}
	}
	return &instanceCreation{request: request, projectID: projectID}, nil
}

// check refuses what the form could not state or the create would refuse, so
// the user is never asked about an instance that cannot be added.
func (c *instanceCreation) check(link string) error {
	if field := unlistedField(c.request.ProtoReflect(), "", instanceRequestFields); field != "" {
		return &toolError{
			Code:       "FIELD_NOT_SUPPORTED",
			Message:    fmt.Sprintf("the request sets %s, which administer_workspace does not take", field),
			Suggestion: "send parent, instanceId and instance with title, engine, environment, activation and one dataSources entry with id, type, authenticationType, host, port, username, password, database, useSsl and verifyTlsCertificate only; set up anything else signed in to the Bytebase console: " + link,
		}
	}
	instance := c.request.GetInstance()
	if engine := instance.GetEngine(); engine != v1pb.Engine_POSTGRES {
		return &toolError{
			Code:       "UNSUPPORTED_CHANGE",
			Message:    fmt.Sprintf("administer_workspace adds PostgreSQL instances, and the request's engine is %s", engine),
			Suggestion: "add an instance of another engine signed in to the Bytebase console: " + link,
		}
	}
	if n := len(instance.GetDataSources()); n != 1 {
		return &toolError{
			Code:       "INVALID_REQUEST",
			Message:    fmt.Sprintf("the instance has %d data sources, and administer_workspace adds an instance with its admin data source alone", n),
			Suggestion: "send one dataSources entry with type ADMIN; add a read-only data source signed in to the Bytebase console: " + link,
		}
	}
	if auth := c.dataSource().GetAuthenticationType(); auth != v1pb.DataSource_AUTHENTICATION_UNSPECIFIED && auth != v1pb.DataSource_PASSWORD {
		return &toolError{
			Code:       "UNSUPPORTED_CHANGE",
			Message:    fmt.Sprintf("administer_workspace adds an instance that signs in with a user name and password, and the request signs in with %s", auth),
			Suggestion: "set up cloud sign-in signed in to the Bytebase console: " + link,
		}
	}
	if why, suggestion := c.invalidValue(); why != "" {
		return &toolError{Code: "INVALID_REQUEST", Message: why, Suggestion: suggestion}
	}
	return nil
}

// invalidValue returns why a value the create needs is missing or malformed,
// and what to send instead, or "".
func (c *instanceCreation) invalidValue() (why, suggestion string) {
	instance, ds := c.request.GetInstance(), c.dataSource()
	_, environmentErr := common.GetEnvironmentID(instance.GetEnvironment())
	switch {
	case !common.IsValidResourceID(c.request.GetInstanceId()):
		return fmt.Sprintf("the instance ID %q is not valid", c.request.GetInstanceId()),
			`use 1 to 63 lowercase letters, digits and dashes, starting with a letter and not ending with a dash, such as "hr-staging"`
	case strings.TrimSpace(instance.GetTitle()) == "":
		return "the instance has no title", `give it the title the console lists it by, such as "HR Staging"`
	case instance.GetEnvironment() != "" && environmentErr != nil:
		return fmt.Sprintf("the environment %q is not an environment's name", instance.GetEnvironment()),
			`send it as "environments/{id}", such as "environments/test", or leave it out for no environment`
	case ds.GetType() != v1pb.DataSourceType_ADMIN:
		return fmt.Sprintf("the data source's type is %s, and an instance is added with its ADMIN data source", ds.GetType()), "set the data source's type to ADMIN"
	case ds.GetId() == "":
		return "the data source has no id", "give the data source a new UUID as its id"
	case strings.TrimSpace(ds.GetHost()) == "":
		return "the data source has no host", "send the server's host name or IP address"
	case !validPort(ds.GetPort()):
		return fmt.Sprintf("the port %q is not a number from 1 to 65535", ds.GetPort()), `send the server's port, such as "5432"`
	case ds.GetUsername() == "":
		return "the data source has no username", "send the user Bytebase signs in as"
	default:
		return "", ""
	}
}

// validPort reports whether the PostgreSQL driver reads port as a port: digits
// in base 10, from 1 to 65535 (parsePort in Bytebase's pgx fork).
func validPort(port string) bool {
	n, err := strconv.ParseUint(port, 10, 16)
	return err == nil && n >= 1
}

// checkEnvironment refuses an environment the workspace does not have. The
// create takes any environment ID, and an instance in a misspelled one escapes
// the rules written for the environment that was meant.
func (s *Server) checkEnvironment(ctx context.Context, c *instanceCreation) error {
	id := c.environmentID()
	if id == "" {
		return nil
	}
	resp, err := s.apiRequest(ctx, v1connect.SettingServiceGetSettingProcedure, map[string]any{"name": common.SettingNamePrefix + v1pb.Setting_ENVIRONMENT.String()})
	if err != nil {
		return errors.Wrap(err, "failed to read the environments")
	}
	var names []string
	switch {
	// Every workspace stores its environments when it is created, so a
	// workspace without the setting has none, as store.GetEnvironment reads it.
	case resp.Status == http.StatusNotFound:
	// A role that may add instances without reading the environments still can.
	case resp.Status == http.StatusForbidden:
		return nil
	case resp.Status >= http.StatusBadRequest:
		return errors.Errorf("failed to read the environments: HTTP %d: %s", resp.Status, parseError(resp.Body))
	default:
		setting := &v1pb.Setting{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(resp.Body, setting); err != nil {
			return errors.Wrap(err, "failed to parse the environments")
		}
		for _, environment := range setting.GetValue().GetEnvironment().GetEnvironments() {
			names = append(names, common.EnvironmentNamePrefix+environment.GetId())
		}
	}
	if slices.Contains(names, common.EnvironmentNamePrefix+id) {
		return nil
	}
	known := "the workspace has no environments"
	if len(names) > 0 {
		known = "the workspace's environments are " + quoteAll(names)
	}
	return &toolError{
		Code:       "INVALID_REQUEST",
		Message:    fmt.Sprintf("the environment %q does not exist, so no instance was added", c.request.GetInstance().GetEnvironment()),
		Suggestion: known + "; use one of them, or leave instance.environment out for no environment",
	}
}

// instanceForm builds the form that asks to add the instance. The password
// appears nowhere in it.
func instanceForm(ctx context.Context, c *instanceCreation, maskingSkips bool, link string) (*adminForm, error) {
	// Activation can take one of the license's fixed number of instance
	// licenses (InstanceService.checkActivationLimit), so the question names it.
	add := "Add instance"
	if c.request.GetInstance().GetActivation() {
		add = "Add activated instance"
	}
	question := func(instanceID string) string {
		return fmt.Sprintf("%s %q to %q?", add, instanceID, c.projectID)
	}
	// Only the account's email may run past a line, so the IDs must fit.
	if lineColumns(question(c.request.GetInstanceId())) > messageLineWidth {
		suggestion := "add the instance signed in to the Bytebase console: " + link
		if lineColumns(question("a")) <= messageLineWidth {
			suggestion = "use a shorter instance ID, or " + suggestion
		}
		return nil, &toolError{
			Code:       "CHANGE_TOO_LONG",
			Message:    fmt.Sprintf("the question naming instance %q and project %q is wider than one line of the confirmation form, so no instance was added", c.request.GetInstanceId(), c.projectID),
			Suggestion: suggestion,
		}
	}
	ds := c.dataSource()
	placement := "No environment."
	if id := c.environmentID(); id != "" {
		placement = fmt.Sprintf("Environment %q.", id)
	}
	changes := []formChange{
		{text: fmt.Sprintf("Host %q, port %s, user %q, %s", ds.GetHost(), ds.GetPort(), ds.GetUsername(), tlsMode(ds))},
		{text: placement + " Its databases join the project for good."},
	}
	if maskingSkips && !c.request.GetInstance().GetActivation() {
		changes = append(changes, formChange{text: instanceNotActivated})
	}
	return fitForm(ctx, strings.TrimSuffix(question(c.request.GetInstanceId()), "?"), changes, "", "", instanceDescription(c), func(needed int, _ string) error {
		return &toolError{
			Code:       "CHANGE_TOO_LONG",
			Message:    fmt.Sprintf("this instance needs %d lines, more than the %d one confirmation form shows, so no instance was added", needed, messageLines),
			Suggestion: "an instance cannot be added in steps, so add it signed in to the Bytebase console: " + link,
		}
	})
}

// tlsMode names the TLS mode the PostgreSQL driver uses: sslmode verify-full or
// require when use_ssl is set (util.GetPGSSLMode, with no CA or SSH tunnel,
// which the entry refuses), and pgx's default, prefer, otherwise, which tries
// TLS and accepts plain text.
func tlsMode(ds *v1pb.DataSource) string {
	switch {
	case !ds.GetUseSsl():
		return "TLS not required"
	case ds.GetVerifyTlsCertificate():
		return "TLS required and verified"
	default:
		return "TLS required, not verified"
	}
}

// instanceDescription is what the message leaves out.
func instanceDescription(c *instanceCreation) string {
	ds := c.dataSource()
	var sb strings.Builder
	fmt.Fprintf(&sb, "PostgreSQL instance %q, titled %q.", c.request.GetInstanceId(), c.request.GetInstance().GetTitle())
	sb.WriteString("\nAfter you apply, Bytebase tests the connection and adds nothing if it fails.")
	if ds.GetPassword() == "" {
		fmt.Fprintf(&sb, " It signs in as %q with no password.", ds.GetUsername())
	} else {
		fmt.Fprintf(&sb, " It signs in as %q with the password in the request; this form does not show it.", ds.GetUsername())
	}
	if database := ds.GetDatabase(); database != "" {
		fmt.Fprintf(&sb, " It connects to database %q, and syncs every database on the server.", database)
	}
	return sb.String()
}

// instanceAddedText reports the added instance: whether Bytebase synced it, and
// whether masking skips it. The create's response sets last_sync_time only when
// the instance sync succeeded, and activation when the instance is activated or
// the license covers every instance.
func (s *Server) instanceAddedText(ctx context.Context, c *instanceCreation, created *v1pb.Instance, maskingSkips bool) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Added: instance %q in project %q", c.request.GetInstanceId(), c.projectID)
	if id := c.environmentID(); id != "" {
		fmt.Fprintf(&sb, ", environment %q", id)
	}
	sb.WriteString(".")
	if created.GetLastSyncTime() != nil {
		sb.WriteString(" Bytebase synced its databases into the project.")
	} else {
		sb.WriteString(" Bytebase has not synced its databases yet; the instance page shows the sync.")
	}
	if maskingSkips && !created.GetActivation() {
		sb.WriteString(" It is not activated, so masking skips it.")
	}
	fmt.Fprintf(&sb, "\nInstance: %s", s.consoleLink(ctx, c.confirmedResource()))
	return sb.String()
}
