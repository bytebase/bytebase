package parsercontext

import (
	"context"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
	oracledriver "github.com/bytebase/omni/oracle/driver"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

// dataSourceReachesLinkTarget reports whether a data source connects to the
// database the link names: its host and port equal one of the link's addresses
// and it connects by the same identity, service name or SID. go-ora connects
// by SID whenever the data source sets one, else by service name, so that is
// the identity compared. Compared whole: 10.0.0.5 is not 10.0.0.50, and ORCL is
// not ORCL.corp.com.
func dataSourceReachesLinkTarget(target oracledriver.ConnectTarget, dataSource *storepb.DataSource) bool {
	host := strings.ToLower(strings.TrimSpace(dataSource.GetHost()))
	port := dataSource.GetPort()
	reachable := false
	for _, endpoint := range target.Endpoints {
		if host == endpoint.Host && port == endpoint.Port {
			reachable = true
			break
		}
	}
	if !reachable {
		return false
	}
	if sid := dataSource.GetSid(); sid != "" {
		return target.SID != "" && strings.EqualFold(sid, target.SID)
	}
	serviceName := dataSource.GetServiceName()
	return serviceName != "" && target.Service != "" && strings.EqualFold(serviceName, target.Service)
}

// selectLinkDefinition picks the link the statement names out of the
// definitions synced from ALL_DB_LINKS, comparing names case-insensitively.
// Oracle appends the domain of the database's global name to `@REMOTE` and
// looks up that name only: `REMOTE` without a domain, `REMOTE.WORLD` under
// the domain WORLD. The domain is not synced, so the name, its domained
// forms and a public and private link of one name are all candidates, and
// they resolve only when they agree on host and user; otherwise the
// statement could reach either, and nothing is returned. A connection
// qualifier follows the domain (`REMOTE.WORLD@Q`) and names a different
// link, so it must match. found reports whether any definition carried the
// name.
func selectLinkDefinition(name string, links []*metadatapb.LinkedDatabaseMetadata) (link *metadatapb.LinkedDatabaseMetadata, found bool) {
	base, qualifier, _ := strings.Cut(name, "@")
	for _, candidate := range links {
		candidateBase, candidateQualifier, _ := strings.Cut(candidate.GetName(), "@")
		if !strings.EqualFold(candidateQualifier, qualifier) {
			continue
		}
		domained := len(candidateBase) > len(base)+1 && strings.EqualFold(candidateBase[:len(base)+1], base+".")
		if !domained && !strings.EqualFold(candidateBase, base) {
			continue
		}
		found = true
		if link == nil {
			link = candidate
			continue
		}
		if !strings.EqualFold(link.GetHost(), candidate.GetHost()) || !strings.EqualFold(link.GetUsername(), candidate.GetUsername()) {
			return nil, true
		}
	}
	return link, found
}

// oracleLinkResolution is the database an Oracle link resolves to. Meta is
// nil when it resolves to nothing, and Reason says why.
type oracleLinkResolution struct {
	InstanceID   string
	DatabaseName string
	Meta         *model.DatabaseMetadata
	Reason       string
}

// resolveOracleLink maps a database link named in a statement run against
// instanceID to the Bytebase database it reaches. The result may decide which
// grants apply to a linked table, so it never guesses: the link's connect
// string must name one service at addresses that a data source of exactly one
// candidate database matches exactly.
func resolveOracleLink(ctx context.Context, s *store.Store, workspaceID string, engine storepb.Engine, instanceID, linkName, schemaName string) (oracleLinkResolution, error) {
	databases, err := s.ListDatabases(ctx, &store.FindDatabaseMessage{
		Workspace:  workspaceID,
		InstanceID: &instanceID,
	})
	if err != nil {
		return oracleLinkResolution{}, err
	}
	// Every database of the instance synced the same ALL_DB_LINKS rows, so the
	// first one that names the link has every definition of it.
	var link *metadatapb.LinkedDatabaseMetadata
	for _, database := range databases {
		meta, err := s.GetDBSchema(ctx, &store.FindDBSchemaMessage{
			Workspace:    workspaceID,
			InstanceID:   database.InstanceID,
			DatabaseName: database.DatabaseName,
		})
		if err != nil {
			return oracleLinkResolution{}, err
		}
		if meta == nil {
			continue
		}
		candidate, found := selectLinkDefinition(linkName, meta.GetProto().GetLinkedDatabases())
		if !found {
			continue
		}
		if candidate == nil {
			return oracleLinkResolution{Reason: "the link is defined more than once with different targets"}, nil
		}
		link = candidate
		break
	}
	if link == nil {
		return oracleLinkResolution{Reason: "no synced database of the instance defines the link"}, nil
	}
	target, ok := oracledriver.ParseConnectString(link.GetHost())
	if !ok {
		return oracleLinkResolution{Reason: "the link's connect string names no single service and host"}, nil
	}
	// The remote schema is the one the statement wrote, else the link's connect user.
	databaseName := link.GetUsername()
	if schemaName != "" {
		databaseName = schemaName
	}
	if databaseName == "" {
		return oracleLinkResolution{Reason: "the link has no connect user and the statement names no schema"}, nil
	}
	candidates, err := s.ListDatabases(ctx, &store.FindDatabaseMessage{
		Workspace:    workspaceID,
		DatabaseName: &databaseName,
		Engine:       &engine,
	})
	if err != nil {
		return oracleLinkResolution{}, err
	}
	var matches []*store.DatabaseMessage
	for _, database := range candidates {
		instance, err := s.GetInstance(ctx, &store.FindInstanceMessage{Workspace: workspaceID, ResourceID: &database.InstanceID})
		if err != nil {
			return oracleLinkResolution{}, err
		}
		if instance == nil {
			continue
		}
		for _, dataSource := range instance.Metadata.GetDataSources() {
			if dataSourceReachesLinkTarget(target, dataSource) {
				matches = append(matches, database)
				break
			}
		}
	}
	switch len(matches) {
	case 0:
		return oracleLinkResolution{Reason: "no data source of any instance holding that database reaches the link's host, port and service"}, nil
	case 1:
	default:
		return oracleLinkResolution{Reason: "data sources of several instances reach the link"}, nil
	}
	linkedDatabase := matches[0]
	meta, err := s.GetDBSchema(ctx, &store.FindDBSchemaMessage{
		Workspace:    workspaceID,
		InstanceID:   linkedDatabase.InstanceID,
		DatabaseName: linkedDatabase.DatabaseName,
	})
	if err != nil {
		return oracleLinkResolution{}, err
	}
	if meta == nil {
		return oracleLinkResolution{Reason: "the linked database has not been synced"}, nil
	}
	return oracleLinkResolution{InstanceID: linkedDatabase.InstanceID, DatabaseName: linkedDatabase.DatabaseName, Meta: meta}, nil
}
