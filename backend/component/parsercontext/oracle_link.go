package parsercontext

import (
	"context"
	"regexp"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
)

// oracleEndpoint is one listener address a database link may connect to.
type oracleEndpoint struct {
	host string
	port string
}

// oracleLinkTarget is what an Oracle database link's connect string
// (ALL_DB_LINKS.HOST) names: one service or SID, reachable at one or more
// addresses. DEFER: every address of one descriptor is taken as the same
// database, as in RAC and Data Guard links, so a descriptor mixing databases
// under one service resolves to whichever address a data source matches;
// upgrade when such a descriptor is seen, then require every address to match.
type oracleLinkTarget struct {
	endpoints []oracleEndpoint
	// Exactly one of service and sid is set, as written. They name different
	// things on a listener and are never matched against each other.
	service string
	sid     string
}

const oracleDefaultPort = "1521"

var (
	tnsKeywordPattern = regexp.MustCompile(`(?i)\(\s*(HOST|PORT|SERVICE_NAME|SID)\s*=\s*([^()]+?)\s*\)`)
	tnsAddressPattern = regexp.MustCompile(`(?i)\(\s*ADDRESS\s*=`)
	// tnsSourceRoutePattern matches a descriptor whose addresses are a hop chain
	// through Connection Manager rather than alternatives.
	tnsSourceRoutePattern = regexp.MustCompile(`(?i)\(\s*SOURCE_ROUTE\s*=\s*(YES|ON|TRUE)\s*\)`)
	// connectSchemePattern matches the Easy Connect Plus `tcp://` / `tcps://` prefix.
	connectSchemePattern = regexp.MustCompile(`(?i)^[a-z]+://`)
)

// parseOracleLinkTarget parses an Oracle connect string into the endpoints and
// the one service it names. Accepted, per the Net Services Administrator's
// Guide: Easy Connect Plus `[tcps://][//]host1[,host2][:port1][,host3:port2]/service[:server][/instance][?params]`,
// where a port applies to the hosts listed before it, and a connect descriptor
// whose ADDRESSes are alternatives for one service (RAC address lists, Data
// Guard description lists). ok=false for anything that names no single
// database: a TNS alias, a descriptor with no host or with several services,
// or a SOURCE_ROUTE hop chain, whose first address is a proxy.
func parseOracleLinkTarget(connectString string) (oracleLinkTarget, bool) {
	s := strings.TrimSpace(connectString)
	if s == "" {
		return oracleLinkTarget{}, false
	}
	if strings.HasPrefix(s, "(") {
		return parseTNSDescriptor(s)
	}
	return parseEasyConnect(s)
}

func parseEasyConnect(s string) (oracleLinkTarget, bool) {
	s = connectSchemePattern.ReplaceAllString(s, "")
	s = strings.TrimPrefix(s, "//")
	hostPart, rest, found := strings.Cut(s, "/")
	if !found || hostPart == "" {
		return oracleLinkTarget{}, false
	}
	service := rest
	if i := strings.IndexAny(service, ":/?"); i >= 0 {
		service = service[:i]
	}
	service = strings.TrimSpace(service)
	if service == "" || strings.ContainsAny(hostPart, " \t") {
		return oracleLinkTarget{}, false
	}
	endpoints, ok := parseEasyConnectHosts(hostPart)
	if !ok {
		return oracleLinkTarget{}, false
	}
	// Easy Connect names a service; a SID needs a descriptor.
	return oracleLinkTarget{endpoints: endpoints, service: service}, true
}

// parseEasyConnectHosts splits `host1[,host2][:port1][,host3:port2]`. A port
// belongs to every host listed since the previous port; hosts with no port
// after them get Oracle's default.
func parseEasyConnectHosts(hostPart string) ([]oracleEndpoint, bool) {
	var endpoints []oracleEndpoint
	unported := 0
	for _, item := range strings.Split(hostPart, ",") {
		host, port, hasPort, ok := splitHostPort(item)
		if !ok {
			return nil, false
		}
		endpoints = append(endpoints, oracleEndpoint{host: strings.ToLower(host), port: oracleDefaultPort})
		unported++
		if hasPort {
			for i := len(endpoints) - unported; i < len(endpoints); i++ {
				endpoints[i].port = port
			}
			unported = 0
		}
	}
	return endpoints, true
}

// splitHostPort splits `host[:port]`, with `[addr]:port` for IPv6.
func splitHostPort(s string) (host, port string, hasPort, ok bool) {
	host = s
	if strings.HasPrefix(s, "[") {
		end := strings.Index(s, "]")
		if end < 0 {
			return "", "", false, false
		}
		host = s[1:end]
		if tail := s[end+1:]; tail != "" {
			if !strings.HasPrefix(tail, ":") {
				return "", "", false, false
			}
			port, hasPort = tail[1:], true
		}
	} else if h, p, found := strings.Cut(s, ":"); found {
		host, port, hasPort = h, p, true
	}
	if host == "" || (hasPort && port == "") {
		return "", "", false, false
	}
	return host, port, hasPort, true
}

func parseTNSDescriptor(s string) (oracleLinkTarget, bool) {
	if tnsSourceRoutePattern.MatchString(s) {
		return oracleLinkTarget{}, false
	}
	// A DESCRIPTION_LIST repeats CONNECT_DATA per description; the same identity
	// named twice is still one. A service and a SID, or two different values,
	// name two databases.
	var identities []string
	target := oracleLinkTarget{}
	for _, m := range tnsKeywordPattern.FindAllStringSubmatch(s, -1) {
		key := strings.ToUpper(m[1])
		if key != "SERVICE_NAME" && key != "SID" {
			continue
		}
		if len(identities) == 0 {
			identities = append(identities, key+"="+m[2])
			if key == "SID" {
				target.sid = m[2]
			} else {
				target.service = m[2]
			}
			continue
		}
		if !strings.EqualFold(identities[0], key+"="+m[2]) {
			identities = append(identities, key+"="+m[2])
		}
	}
	if len(identities) != 1 {
		return oracleLinkTarget{}, false
	}
	var endpoints []oracleEndpoint
	for _, loc := range tnsAddressPattern.FindAllStringIndex(s, -1) {
		block := tnsBlock(s, loc[0])
		var hosts, ports []string
		for _, m := range tnsKeywordPattern.FindAllStringSubmatch(block, -1) {
			switch strings.ToUpper(m[1]) {
			case "HOST":
				hosts = append(hosts, m[2])
			case "PORT":
				ports = append(ports, m[2])
			default:
			}
		}
		if len(hosts) != 1 || len(ports) > 1 {
			return oracleLinkTarget{}, false
		}
		port := oracleDefaultPort
		if len(ports) == 1 {
			port = ports[0]
		}
		endpoints = append(endpoints, oracleEndpoint{host: strings.ToLower(hosts[0]), port: port})
	}
	if len(endpoints) == 0 {
		return oracleLinkTarget{}, false
	}
	target.endpoints = endpoints
	return target, true
}

// tnsBlock returns the parenthesized block that starts at s[start].
func tnsBlock(s string, start int) string {
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		default:
		}
	}
	return s[start:]
}

// dataSourceReachesLinkTarget reports whether a data source connects to the
// database the link names: its host and port equal one of the link's addresses
// and it connects by the same identity, service name or SID. go-ora connects
// by SID whenever the data source sets one, else by service name, so that is
// the identity compared. Compared whole: 10.0.0.5 is not 10.0.0.50, and ORCL is
// not ORCL.corp.com.
func dataSourceReachesLinkTarget(target oracleLinkTarget, dataSource *storepb.DataSource) bool {
	host := strings.ToLower(strings.TrimSpace(dataSource.GetHost()))
	port := dataSource.GetPort()
	reachable := false
	for _, endpoint := range target.endpoints {
		if host == endpoint.host && port == endpoint.port {
			reachable = true
			break
		}
	}
	if !reachable {
		return false
	}
	if sid := dataSource.GetSid(); sid != "" {
		return target.sid != "" && strings.EqualFold(sid, target.sid)
	}
	serviceName := dataSource.GetServiceName()
	return serviceName != "" && target.service != "" && strings.EqualFold(serviceName, target.service)
}

// selectLinkDefinition picks the link the statement names out of the
// definitions synced from ALL_DB_LINKS, comparing names case-insensitively.
// An exact name match wins over a domained one: Oracle stores a link created
// in a database with DB_DOMAIN set as `REMOTE.WORLD` and resolves `@REMOTE`
// to it by appending the domain. Several definitions of one name (a public
// and a private link, two domains) resolve only when they agree on host and
// user; otherwise the statement could reach either, and nothing is returned.
// found reports whether any definition carried the name.
func selectLinkDefinition(name string, links []*metadatapb.LinkedDatabaseMetadata) (link *metadatapb.LinkedDatabaseMetadata, found bool) {
	var exact, prefixed []*metadatapb.LinkedDatabaseMetadata
	for _, candidate := range links {
		switch {
		case strings.EqualFold(candidate.GetName(), name):
			exact = append(exact, candidate)
		case len(candidate.GetName()) > len(name)+1 && strings.EqualFold(candidate.GetName()[:len(name)+1], name+"."):
			prefixed = append(prefixed, candidate)
		default:
		}
	}
	candidates := exact
	if len(candidates) == 0 {
		candidates = prefixed
	}
	for _, candidate := range candidates {
		if link == nil {
			link = candidate
			continue
		}
		if !strings.EqualFold(link.GetHost(), candidate.GetHost()) || !strings.EqualFold(link.GetUsername(), candidate.GetUsername()) {
			return nil, true
		}
	}
	return link, len(candidates) > 0
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
	target, ok := parseOracleLinkTarget(link.GetHost())
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
