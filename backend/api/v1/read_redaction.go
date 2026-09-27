package v1

import (
	"fmt"
	"regexp"
)

// Read-path redaction for the values a whole field cannot simply be emptied of.
//
// A stored secret must not come back out of a read. The instance converter has
// said so about data-source credentials for a long time — "We don't return the
// password and SSLs on reads" — and where the whole field is the secret, the
// proto says so too: INPUT_ONLY on the field, an OUTPUT_ONLY <field>_set bool
// beside it for the one thing a client still needs to know. A project's chat
// webhook URL is now shaped that way.
//
// What is left here is the case that shape does not fit: an instance role's
// attribute is grant text a DBA reads, with a credential clause buried inside
// it, so the field stays and the clause is masked.
//
// This is not the audit redaction in audit.go. That one masks what gets
// LOGGED; this masks what gets RETURNED to the caller. The values overlap
// because they are the same secrets, but the two run on different paths and
// neither can stand in for the other.

// redactedValue stands in for a secret inside a larger string a read does
// return. Nothing that has to parse a real value will mistake it for one.
const redactedValue = "<redacted>"

// mysqlGrantCredential matches the credential clause SHOW GRANTS writes into a
// grant line on the MySQL family. MariaDB emits every form; MySQL 8.0 and
// 5.7.44 emit none, so this is a no-op there.
//
//	GRANT ... TO `u`@`%` IDENTIFIED BY PASSWORD '*05A6C9A5...'
//	GRANT ... TO `u`@`%` IDENTIFIED VIA ed25519 USING 'ZIgUREUg...'
//	GRANT ... TO `u`@`%` IDENTIFIED VIA ed25519 USING 'A' OR mysql_native_password USING '*B'
//
// The quoted run is matched greedily, to the last quote on the line, and that
// is deliberate. MariaDB does not escape a quote inside the stored auth
// string: an auth string of ab'cd comes back as USING 'ab'cd', which is not
// even re-parseable SQL. A lazy match would stop at the inner quote and return
// the rest of the secret. Greedy can only ever take too much — a REQUIRE
// SUBJECT '<dn>' following the credential goes into the mask with it — and
// taking too much of a grant line is a display cost, where taking too little
// is the leak this exists to close. The clauses that carry no quotes, REQUIRE
// SSL and WITH GRANT OPTION, sit outside the match and survive.
//
// The keyword is required, so an attribute that is not MySQL grant text is
// left alone: PostgreSQL, Redshift and CockroachDB store role keywords,
// Snowflake stores GRANT statements, Elasticsearch stores a role list, and
// none of them can match.
var mysqlGrantCredential = regexp.MustCompile(`(IDENTIFIED (?:BY|VIA) [^'\n]*)'[^\n]*'`)

// redactRoleAttribute masks the credential clause in an instance role's
// attribute text, which on MariaDB carries every database account's password
// hash — enough to clone the account onto another server, and to crack
// offline. The rest of the text is the grant list a DBA reads, so it stays.
func redactRoleAttribute(attribute string) string {
	return mysqlGrantCredential.ReplaceAllString(attribute, "${1}'"+redactedValue+"'")
}

// deliveryStatusCode pulls the HTTP status out of a webhook poster's failure.
//
// That status is the one thing from the message a caller who cannot see the URL
// is allowed to learn, and it is matched rather than the URL removed. The
// difference matters: removing the URL means knowing every rendering the stack
// can produce of it, and three rounds of that missed a different one each time
// — the re-encoded form net/http prints, the form it prints after rewriting
// embedded userinfo, and the scheme-less path it prints when a host answers a
// redirect with a relative Location. The last one is the whole credential for
// Slack, Discord and Feishu, whose secret lives in the path. An allowlist
// cannot lose that race, because a rendering nobody anticipated is not on it.
var deliveryStatusCode = regexp.MustCompile(`status code: (\d{3})`)

// storedWebhookDeliveryFailure describes a failed test post to a URL the caller
// cannot read. The status code says what to do about it — a 404 means the hook
// was revoked and has to be re-created — and nothing else from the poster's
// message comes back.
//
// That costs something, and knowingly. The response body often carries the
// vendor's own reason (invalid_payload, channel_not_found), and three posters
// return the body alone with no URL around it, so those failures come back as
// the generic sentence. Keeping the body would mean trusting arbitrary remote
// text not to contain the request path, which is the bet this function exists
// because we lost.
//
// A caller testing a URL they typed gets the poster's message in full. They
// supplied the URL, so there is nothing to withhold from them, and the full
// diagnosis is what makes the create form's Test button worth pressing.
func storedWebhookDeliveryFailure(err error) string {
	if match := deliveryStatusCode.FindStringSubmatch(err.Error()); match != nil {
		return fmt.Sprintf("the stored webhook URL answered with status code %s", match[1])
	}
	return "the test notification could not be delivered to the stored webhook URL"
}
