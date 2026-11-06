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

const (
	instanceActivated = "Activated: it uses one of the workspace's instance licenses."
	// instanceNotActivated is the form's effect line, so it never moves to the
	// description, which Claude Code's fullscreen renderer cuts to one line:
	// masking skips an instance that is not activated unless the license covers
	// every instance (LicenseService.IsFeatureEnabledForInstance).
	instanceNotActivated = "Not activated, so masking skips it unless every instance is licensed."
)

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

	// Read on both rounds: an answer is acted on only while the project, the
	// instance ID and the environment still allow the instance the form states.
	if err := s.checkInstanceTargets(ctx, change); err != nil {
		return formatToolError(err), nil, nil
	}
	form, err := instanceForm(ctx, change, link)
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
	resp, err := sendInstanceCreation(confirmedCtx, s, test)
	if err != nil {
		return formatToolError(errors.Wrap(err, "failed to test the connection")), nil, nil
	}
	if resp.Status >= http.StatusBadRequest {
		return formatToolError(instanceCreationError(resp, change)), nil, nil
	}
	resp, err = sendInstanceCreation(confirmedCtx, s, change.request)
	if err != nil {
		return formatToolError(errors.Wrap(err, "failed to add the instance")), nil, nil
	}
	if resp.Status >= http.StatusBadRequest {
		return formatToolError(instanceCreationError(resp, change)), nil, nil
	}
	created := &v1pb.Instance{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(resp.Body, created); err != nil {
		return formatToolError(errors.Wrap(err, "the instance was added, but its description could not be read")), nil, nil
	}
	text := s.instanceAddedText(ctx, change, created)
	instanceLink := s.consoleLink(ctx, change.confirmedResource())
	return textResult(text), &AdministerWorkspaceOutput{OperationID: input.OperationID, Changed: true, Message: text, Link: instanceLink}, nil
}

func sendInstanceCreation(ctx context.Context, s *Server, request *v1pb.CreateInstanceRequest) (*apiResponse, error) {
	body, err := protojson.Marshal(request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build the request")
	}
	return s.apiRequest(ctx, v1connect.InstanceServiceCreateInstanceProcedure, json.RawMessage(body))
}

// instanceCreationError reports a refused connection test or create. Only a
// failed connection test sets the connection failure category, so any other
// refusal reads as the handler states it.
func instanceCreationError(resp *apiResponse, change *instanceCreation) error {
	if resp.Headers.Get(common.ConnectionFailureCategoryHeader) == "" {
		return errors.Errorf("failed to add the instance: HTTP %d: %s", resp.Status, parseError(resp.Body))
	}
	ds := change.dataSource()
	return &toolError{
		Code:       "CONNECTION_FAILED",
		Message:    fmt.Sprintf("Bytebase could not connect to %q, port %s, as %q, so no instance was added: %s", ds.GetHost(), ds.GetPort(), ds.GetUsername(), parseError(resp.Body)),
		Suggestion: "check the host, port, user and password with the user, then call administer_workspace again; the user confirms the corrected request in a new form",
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
			Suggestion: "send parent, instanceId and instance with title, engine, environment, activation and one dataSources entry with id, type, host, port, username, password, database, useSsl and verifyTlsCertificate only; set up anything else signed in to the Bytebase console: " + link,
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

// validPort reports whether port is a TCP port number written as the console's
// port field takes it: digits alone.
func validPort(port string) bool {
	n, err := strconv.Atoi(port)
	return err == nil && n >= 1 && n <= 65535 && strconv.Itoa(n) == port
}

// checkInstanceTargets refuses a project that does not exist or is archived, an
// instance ID the project already holds, and an environment that does not
// exist. These are checks for the user's sake: the create stays the authority,
// and refuses an ID taken anywhere else. A read the user's role does not allow
// checks nothing, so a role that may add instances without these reads still
// can.
func (s *Server) checkInstanceTargets(ctx context.Context, c *instanceCreation) error {
	project := common.FormatProject(c.projectID)
	resp, err := s.apiRequest(ctx, v1connect.ProjectServiceGetProjectProcedure, map[string]any{"name": project})
	if err != nil {
		return errors.Wrap(err, "failed to read the project")
	}
	switch {
	case resp.Status == http.StatusNotFound:
		return &toolError{
			Code:       "INVALID_REQUEST",
			Message:    fmt.Sprintf("the project %q does not exist, so no instance was added", project),
			Suggestion: "find the project with call_api ProjectService/SearchProjects, and send its name as parent",
		}
	case readRefused(resp):
	case resp.Status >= http.StatusBadRequest:
		return errors.Errorf("failed to read the project: HTTP %d: %s", resp.Status, parseError(resp.Body))
	case resourceState(resp.Body) == v1pb.State_DELETED.String():
		return &toolError{
			Code:       "INVALID_REQUEST",
			Message:    fmt.Sprintf("the project %q is archived, so no instance was added", project),
			Suggestion: "add the instance to another project, or restore this one signed in to the Bytebase console",
		}
	default:
	}

	name := c.confirmedResource()
	resp, err = s.apiRequest(ctx, v1connect.InstanceServiceGetInstanceProcedure, map[string]any{"name": name})
	if err != nil {
		return errors.Wrap(err, "failed to look up the instance ID")
	}
	switch {
	case resp.Status == http.StatusNotFound, readRefused(resp):
	case resp.Status >= http.StatusBadRequest:
		return errors.Errorf("failed to look up the instance ID: HTTP %d: %s", resp.Status, parseError(resp.Body))
	case resourceState(resp.Body) == v1pb.State_DELETED.String():
		return &toolError{
			Code:       "ALREADY_EXISTS",
			Message:    fmt.Sprintf("a deleted instance in project %q holds the instance ID %q, so no instance was added", c.projectID, c.request.GetInstanceId()),
			Suggestion: "choose another instance ID, or restore that instance signed in to the Bytebase console",
		}
	default:
		return &toolError{
			Code:       "ALREADY_EXISTS",
			Message:    fmt.Sprintf("the instance %q already exists in project %q, so no instance was added", c.request.GetInstanceId(), c.projectID),
			Suggestion: "if it is the server the user means, it is already added: " + s.consoleLink(ctx, name) + ". To add another server, choose another instance ID",
		}
	}
	return s.checkEnvironment(ctx, c)
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
	var ids []string
	switch {
	// Every workspace stores its environments when it is created, so a
	// workspace without the setting has none, as store.GetEnvironment reads it.
	case resp.Status == http.StatusNotFound:
	case readRefused(resp):
		return nil
	case resp.Status >= http.StatusBadRequest:
		return errors.Errorf("failed to read the environments: HTTP %d: %s", resp.Status, parseError(resp.Body))
	default:
		setting := &v1pb.Setting{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(resp.Body, setting); err != nil {
			return errors.Wrap(err, "failed to parse the environments")
		}
		for _, environment := range setting.GetValue().GetEnvironment().GetEnvironments() {
			ids = append(ids, environment.GetId())
		}
	}
	if slices.Contains(ids, id) {
		return nil
	}
	known := "the workspace has no environments"
	if len(ids) > 0 {
		quoted := make([]string, 0, len(ids))
		for _, environment := range ids {
			quoted = append(quoted, fmt.Sprintf("%q", common.EnvironmentNamePrefix+environment))
		}
		known = "the workspace's environments are " + strings.Join(quoted, ", ")
	}
	return &toolError{
		Code:       "INVALID_REQUEST",
		Message:    fmt.Sprintf("the environment %q does not exist, so no instance was added", c.request.GetInstance().GetEnvironment()),
		Suggestion: known + "; use one of them, or leave instance.environment out for no environment",
	}
}

// readRefused reports a read the user's role does not allow.
func readRefused(resp *apiResponse) bool {
	return resp.Status == http.StatusForbidden || resp.Status == http.StatusUnauthorized
}

// resourceState reads the state of a project or an instance from its JSON.
func resourceState(body json.RawMessage) string {
	var resource struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(body, &resource); err != nil {
		return ""
	}
	return resource.State
}

// instanceForm builds the form that asks to add the instance: the connection,
// the environment and the lasting effect on the project, and the activation.
// The password appears nowhere in it.
func instanceForm(ctx context.Context, c *instanceCreation, link string) (*adminForm, error) {
	question := fmt.Sprintf("Add instance %q to project %q", c.request.GetInstanceId(), c.projectID)
	// Only the account's email may run past a line, so the IDs must fit.
	if lineColumns(question+"?") > messageLineWidth {
		return nil, &toolError{
			Code:       "CHANGE_TOO_LONG",
			Message:    fmt.Sprintf("the question naming instance %q and project %q is wider than one line of the confirmation form, so no instance was added", c.request.GetInstanceId(), c.projectID),
			Suggestion: "use a shorter instance ID, or add the instance signed in to the Bytebase console: " + link,
		}
	}
	ds := c.dataSource()
	placement := "No environment."
	if id := c.environmentID(); id != "" {
		placement = fmt.Sprintf("Environment %q.", id)
	}
	changes := []formChange{
		{text: fmt.Sprintf("Host %q, port %s, user %q, %s", ds.GetHost(), ds.GetPort(), ds.GetUsername(), tlsMode(ds)), effect: true},
		{text: placement + " Its databases join the project for good."},
	}
	effect, fixed := "", instanceActivated
	if !c.request.GetInstance().GetActivation() {
		effect, fixed = instanceNotActivated, ""
	}
	return fitForm(ctx, question, changes, effect, fixed, instanceDescription(c), func(needed int, _ string) error {
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

// instanceDescription is what the message leaves out. Claude Code's fullscreen
// renderer shows only its first line.
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

// instanceAddedText reports the added instance: its databases, and whether
// masking skips it. The create's response sets activation when the instance is
// activated or the license covers every instance.
func (s *Server) instanceAddedText(ctx context.Context, c *instanceCreation, created *v1pb.Instance) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Added: instance %q in project %q", c.request.GetInstanceId(), c.projectID)
	if id := c.environmentID(); id != "" {
		fmt.Fprintf(&sb, ", environment %q", id)
	}
	sb.WriteString(".")
	databases, more, err := s.instanceDatabases(ctx, c.confirmedResource())
	switch {
	case err != nil:
		fmt.Fprintf(&sb, " Its databases could not be listed: %v.", err)
	case len(databases) == 0:
		sb.WriteString(" Bytebase lists no database on it yet; the instance page shows its sync.")
	default:
		quoted := make([]string, 0, len(databases))
		for _, database := range databases {
			quoted = append(quoted, fmt.Sprintf("%q", database))
		}
		fmt.Fprintf(&sb, " Its databases: %s", strings.Join(quoted, ", "))
		if more {
			sb.WriteString(" and more")
		}
		sb.WriteString(".")
	}
	if !created.GetActivation() {
		sb.WriteString(" It is not activated, so masking skips it; activate it signed in to the Bytebase console.")
	}
	fmt.Fprintf(&sb, "\nInstance: %s", s.consoleLink(ctx, c.confirmedResource()))
	return sb.String()
}

// instanceDatabases lists the names of the databases Bytebase holds on the
// instance. more reports a list longer than one page, which the tool does not
// read.
func (s *Server) instanceDatabases(ctx context.Context, instance string) ([]string, bool, error) {
	resp, err := s.apiRequest(ctx, v1connect.DatabaseServiceListDatabasesProcedure, map[string]any{"parent": instance, "pageSize": 1000})
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to list the databases")
	}
	if err := checkAPIResponse(resp, "list the instance's databases", "bb.instances.get"); err != nil {
		return nil, false, err
	}
	var found struct {
		Databases []struct {
			Name string `json:"name"`
		} `json:"databases"`
		NextPageToken string `json:"nextPageToken"`
	}
	if err := json.Unmarshal(resp.Body, &found); err != nil {
		return nil, false, errors.Wrap(err, "failed to parse the databases")
	}
	names := make([]string, 0, len(found.Databases))
	for _, database := range found.Databases {
		_, name, _ := strings.Cut(database.Name, "/"+common.DatabaseIDPrefix)
		names = append(names, name)
	}
	return names, found.NextPageToken != "", nil
}
