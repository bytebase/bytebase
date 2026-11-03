package auth

import (
	"maps"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// MCPDenialWording is one reason's sentence plus the class it belongs to. The
// class is carried here because a reason implies one — a mechanism that breaks
// the MCP boundary is what FORBIDDEN means, a scope decision is what EXCLUDED
// means — and one table saying so is what lets the lint check the pairing and
// the gate refuse to print the wrong half of it.
type MCPDenialWording struct {
	Class    v1pb.MCPMethodClass
	Sentence string
}

// mcpDenialReasons is UX copy, NOT the classification and NOT the mapping.
// Both of those live on the RPC itself, as the bytebase.v1.mcp_method_class and
// bytebase.v1.mcp_denial_reason annotations, beside permission / audit /
// auth_method — one source of truth, visible where the RPC is defined. The gate
// reads them off the AuthContext the auth interceptor resolves, and MCP
// discovery reads them off the same descriptors; the table lives here, like
// MCPClassIsRefused, so both give the same reason. It only turns a reason into a
// sentence the agent can act on, so it has one row per mechanism rather than
// one per method, and a missing row costs wording, never enforcement.
//
// Each sentence completes "<procedure> is not available to MCP sessions because
// ___", so it states what the method does rather than merely that it is
// refused: a denial whose stated reason has drifted from the mechanism is worse
// than a bare refusal, because it is the thing the next reader trusts.
var mcpDenialReasons = map[v1pb.MCPDenialReason]MCPDenialWording{
	// The method hands a token back to the caller. For a non-web caller — and
	// an MCP session is always one — finalizeLogin and switchWorkspaceInternal
	// put a plain bb.user.access token in the response body. That token is not
	// audience-bound to the MCP resource, survives revocation of the OAuth
	// grant, and ignores the workspace MCP kill switch, so obtaining one ends
	// the MCP boundary for good.
	v1pb.MCPDenialReason_MINTS_CREDENTIAL: {v1pb.MCPMethodClass_FORBIDDEN, "it hands back a login token that would outlive the MCP grant"},

	// The method drives the out-of-band reset flow — mailing a reset or login
	// code, or consuming one — that sets or delivers the very secret Login
	// accepts. Denying Login alone would leave the agent holding the credential
	// for the next human login.
	v1pb.MCPDenialReason_RESETS_CREDENTIAL: {v1pb.MCPMethodClass_FORBIDDEN, "it drives the credential-reset flow that sets or delivers the secret a login accepts"},

	// UpdateUser's password and MFA branches take no proof of the old password.
	// A caller updating itself needs no permission at all, and on self-hosted a
	// caller holding bb.users.update reaches any other end user's password too
	// (user_service.go: the caller != subject branch checks that permission and
	// nothing further). Either way the session ends up holding credentials it
	// can log in with. The whole method is refused, not just those branches —
	// the classification is per method.
	v1pb.MCPDenialReason_TAKES_OVER_ACCOUNT: {v1pb.MCPMethodClass_FORBIDDEN, "it can rewrite an account's credentials, which would let the session take that account over"},

	// Logout deletes the web refresh token and expires the session cookies. It
	// mints nothing; it destroys the human's own login session, which an agent
	// acting on their behalf has no business doing.
	v1pb.MCPDenialReason_ENDS_SESSION: {v1pb.MCPMethodClass_FORBIDDEN, "it ends the human's own login session"},

	// The workspace-lifecycle pair. Both end in
	// AuthService.switchWorkspaceInternal, which mints a plain workspace token
	// whenever the caller has no refresh cookie — and an MCP session never has
	// one — after having already destroyed the caller's membership.
	v1pb.MCPDenialReason_ENDS_MEMBERSHIP: {v1pb.MCPMethodClass_FORBIDDEN, "it destroys the caller's own workspace membership and mints a plain workspace token"},

	// The method leaves someone holding a principal the caller is not. Four
	// ways, all of them annotated:
	//   - issuing the credential outright — a service key or SCIM token
	//     returned in plaintext, a caller-chosen password on a new account;
	//   - carrying an existing one out — a stored client secret or SMTP
	//     password sent to a host the caller named;
	//   - choosing what will later be trusted to mint one — the issuer and
	//     subject ExchangeToken validates against;
	//   - redirecting where one gets delivered — UpdateEmail rebinds any
	//     account to an address the caller picked, and the reset flow mails the
	//     code to whatever address the account then carries.
	// The levers that contain a runaway session all act on the caller's own
	// principal: revoke the OAuth grant, flip the workspace MCP switch,
	// deactivate the human. None of them reaches a principal that was never the
	// caller's, so what these give away outlives all three.
	//
	// InstanceService/UpdateDataSource is the same "carry an existing one out"
	// shape as the two Test methods, against the database's credentials rather
	// than Bytebase's own: it merges a partial request onto the STORED,
	// already-decrypted data source, so an update_mask naming only a
	// destination field (host, port, ssh_host, additional_addresses,
	// sasl_config.kdc_host) keeps the stored password, ssl_key and
	// ssh_private_key. With validate_only it dials the caller's host
	// immediately and persists nothing; without it the retarget is written and
	// SyncInstance triggers the connection on demand. No allowlist filters the
	// host on either path. A database user is a principal other than the
	// caller, the same way the SMTP account behind TestEmailSetting is.
	//
	// Its siblings are NOT here and the line is severity, not tidiness:
	// UpdateInstance and BatchUpdateInstances rebuild the data-source list
	// wholesale, so every secret is wiped unless resent — except the Kerberos
	// keytab, which retainStoredKeytabs used to inherit by data-source ID.
	// That narrower, persist-only version of the vector is now closed at the
	// retention rule instead of at reachability: a keytab is not inherited to
	// a destination the caller moved (instance_service_converter.go). It costs
	// an agent no instance management, something no one has yet decided it
	// should lose, and it binds human callers too — a Kerberos host edit now
	// asks for the keytab again, on the console as much as on the API.
	// The keytab is the ONLY secret that rule reaches, because it is the only
	// one those two methods inherit. On UpdateDataSource above, every other
	// stored secret still rides an update_mask that names only a destination,
	// which is why that method is refused here rather than fixed there:
	// requiring a password re-supplied on every host edit is a product call.
	// AddDataSource and CreateInstance also dial on validate_only but build
	// the data source entirely from the request, so they carry no stored
	// secret and are not this class. BOT-57.
	//
	// The Undelete* family (user, service account, workload identity) is
	// deliberately NOT in this group, and it is the nearest thing left out, so
	// the reasoning is recorded rather than left to be rediscovered. It
	// restores a principal whose password or key hash survived the soft
	// delete, so an operator's deactivation is undone. It is out because the
	// caller learns and chooses nothing: the credential goes back to whoever
	// already had it, and a second delete takes it away again. Issuing beats
	// re-arming, and this mechanism is about issuing. BOT-54.
	v1pb.MCPDenialReason_MINTS_CREDENTIAL_FOR_OTHERS: {v1pb.MCPMethodClass_FORBIDDEN, "it hands someone control of a principal other than the caller, which revoking this session would not take back"},

	// No method carries this now. It was SettingService/UpdateSetting's, which
	// writes the MCP ceiling (value.mcp.capability), the SMTP relay that mails
	// password resets, the AI endpoint the stored key is sent to, and the
	// sign-in switches. UpdateSetting is WRITE, and the gate serves it only with
	// a confirmation for the setting it names (mcpRequestShapeRefusals), which
	// administer_workspace attaches for the approval rules alone. The value stays
	// for the next method that rewrites the session's boundary.
	v1pb.MCPDenialReason_REWRITES_SESSION_BOUNDARY: {v1pb.MCPMethodClass_FORBIDDEN, "it rewrites the workspace settings that bound this session, including the switch meant to contain it"},

	v1pb.MCPDenialReason_DRIVES_THE_APPROVAL_DECISION: {v1pb.MCPMethodClass_FORBIDDEN, "it can let a change or an access request pass its approval without an approver deciding it"},

	// The other refused class. An EXCLUDED method is out of scope for the modes
	// this release ships, and an admin-capable ceiling could legitimately serve
	// it one day; a FORBIDDEN method breaks the MCP boundary, so no ceiling
	// serves it. A denial that blurred the two would tell an operator the wrong
	// thing about whether asking is worth it. The class on each row is what
	// keeps them apart.
	v1pb.MCPDenialReason_ADMINISTERS_THE_WORKSPACE:   {v1pb.MCPMethodClass_EXCLUDED, "it administers the workspace rather than doing database work, and no MCP mode this release ships covers workspace administration"},
	v1pb.MCPDenialReason_READS_OTHER_USERS_SQL:       {v1pb.MCPMethodClass_EXCLUDED, "it returns SQL that other people wrote, across the workspace or past the sharing that keeps a saved query private"},
	v1pb.MCPDenialReason_OPENS_AN_ADMIN_CONNECTION:   {v1pb.MCPMethodClass_EXCLUDED, "it opens an admin-credentialed connection to the database and returns other sessions' live, unmasked SQL"},
	v1pb.MCPDenialReason_SENDS_DATA_TO_A_THIRD_PARTY: {v1pb.MCPMethodClass_EXCLUDED, "it spends a stored workspace credential to send whatever the caller passes to a third party"},
	v1pb.MCPDenialReason_REDRAWS_GOVERNANCE_BOUNDARY: {v1pb.MCPMethodClass_EXCLUDED, "it changes which rules govern a database or which databases a change that targets a database group runs on, and an agent does not choose the checks on its own changes"},
	// No method carries this one: the three leaks it was written for are
	// redacted on the read path and their eight methods are READ. The row
	// stays so the next read found leaking is one annotation away from a
	// denial that explains itself, rather than two edits away.
	v1pb.MCPDenialReason_RETURNS_A_STORED_SECRET: {v1pb.MCPMethodClass_EXCLUDED, "its response carries a stored secret that the product redacts everywhere else"},
}

// The fallback wording, per class, for a method whose reason is unset, unknown
// to this build, or recorded for the other refused class. The class annotation
// is what denies; the table only supplies a better sentence.
const (
	reasonForbiddenClass = "it is not reachable by an AI agent session"
	reasonExcludedClass  = "no MCP mode this release ships serves it"
)

// MCPDenialReasons returns the wording table, for the lint that holds every
// annotated reason against it.
func MCPDenialReasons() map[v1pb.MCPDenialReason]MCPDenialWording {
	return maps.Clone(mcpDenialReasons)
}

// MCPDenialSentence is the sentence for a refused method's recorded reason, or
// its class's fallback.
//
// The class is checked here as well as in the lint, and that is the point of
// carrying it on the row. One enum means a reason recorded for the wrong class
// is a value, not a parse error, so the runtime has to decline it: a FORBIDDEN
// method must never explain itself with an exclusion's sentence. A mismatch
// costs wording, the way a missing row does — never enforcement.
func MCPDenialSentence(class v1pb.MCPMethodClass, reason v1pb.MCPDenialReason) string {
	if wording, ok := mcpDenialReasons[reason]; ok && wording.Class == class {
		return wording.Sentence
	}
	if class == v1pb.MCPMethodClass_FORBIDDEN {
		return reasonForbiddenClass
	}
	return reasonExcludedClass
}
