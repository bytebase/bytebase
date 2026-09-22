# Audit: What Produces a Row

Author: Vincent Huang + AI
Last Update: Sept. 17, 2026
Reviewers: [x] Danny Xu [x] Edward Lu [x] Xzavier Zane
Related Docs: [Draft 3](https://docs.google.com/document/d/1zBam0DT3jtLrDUnSIdiUp2oz3uLdKx53Jb13FGvy_LI/edit), [Audit logs in comparable products](https://claude.ai/code/artifact/579af5ff-ff73-41ad-b446-9c94e9da7b46), [Audit compliance](https://docs.google.com/presentation/d/1CyG-HsnhclncnAO0JlxE4Kit8tr5ZInKHsA8CLvPDC8/edit?usp=sharing)

## Background

- Two interceptor chains serve the same handlers, and the first-listed
  interceptor is the outermost one:

```
public chain:    validate -> auth -> ACL -> audit -> handler
internal MCP:    validate -> auth -> audit -> gate -> ACL -> handler
```

- On the public chain a permission denial returns before the audit
  interceptor runs and leaves no trace. On the MCP chain the same denial
  writes a row. Nothing decided it that way.
- The audit trigger also fires when the MCP gate refused the call, through a
  runtime flag, so a method with no audit annotation can produce rows and the
  protos no longer say what is stored.
- A permission check inside a handler on an audited method writes a row
  today, with the error as its status, and every failed login writes a row.
- The audit log lives in the metadata database, where audit writes compete
  with the operational workload. Stdout is a workspace setting, TEAM and
  above, self-hosted only. Today it mirrors the database, and Bytebase Cloud
  refuses it.

## Goals

- **Give the compliance reader a record of refused calls.** The standards
  in the related deck require a log of refused access attempts. A call
  refused before it reaches a handler leaves no record today.
- **Keep the console the administrator's record.** It stores exactly what it
  stores today, and nothing that reader does not need.
- **Audit a denied call the same way on both chains,** human or agent.
- **Follow what comparable products do:** a stored record for admins and a
  stream for compliance [7].

## 1. Two readers, two sinks

Two kinds of reader open the audit log, and they want different things.

An administrator opens the console after something went wrong: a row was
dropped, a schema changed, a setting moved. They want to know who did it,
from where, and with what request, and they want it in the product, beside
the object it touched. A refused call is not on that list. It changed
nothing, and an engineer who needs a permission asks for it rather than
being discovered in a log.

A compliance reader wants the opposite: every access decision, refused ones
most of all, kept for a fixed period in a system of record they control and
reviewed with tools built for that. They do not read it in the console. They
read it in a SIEM, and they want it complete more than they want it
convenient.

The database serves the first reader, so it stores what an admitted caller
did. The stream serves the second, so it carries everything, refusals
included, and leaves storage and retention to the reader's own system. One
sink for both would fail one of them: everything in the database puts
compliance volume on the metadata store, and everything in the stream takes
the administrator's record out of the product. The rest of this doc follows
from that split.

We should keep two sinks with two jobs: the database for the administrator's
record, stdout for the compliance record.

```
store  = audited method and the call reached its handler
stream = stdout on and (stored or refused by a permission check)
```

| Call | Database | Stdout, when on |
|---|---|---|
| Audited method, reached the handler, any outcome | stored | streamed |
| Any method, refused by the permission check at the interceptor | not stored | streamed |
| Unaudited method, refused by the permission check inside the handler | not stored | streamed |
| Unaudited method, reached the handler, not refused | not stored | not streamed |

- The database rule is today's rule, because today's rule already describes
  what the administrator reads: what an admitted caller did on a method worth
  recording. A permission refusal is streamed wherever it is made, and stored
  only when that rule already stores the call, so the database stays
  identical to today. GCP and Azure keep the same
  always-on tier of administrative changes with reads off by default [1][5];
  AWS logs management reads and writes by default and makes data events
  opt-in [3][4].
- Stdout excludes nothing beyond what the row itself excludes: redaction of
  sensitive fields, and successful validate-only calls.
- What the stream guarantees. Bytebase writes each line to its own standard
  output once, synchronously, before the database insert, so a failed insert
  does not lose the line. It does not buffer, retry or confirm delivery, and
  it cannot see a failed write to stdout; the writer is best-effort today and
  stays so. Collection, durability and retention are the operator's log
  pipeline, and a line the pipeline drops is gone.
- Refused means refused by a permission check. A rejected credential is
  turned away before audit runs, and the one streaming call writes on send,
  so a refusal on receive reaches neither sink. Ownership and workflow rules
  are not permission checks (section 3).
- Stdout is self-hosted, TEAM and above, and Bytebase Cloud refuses it, so
  this change adds no refusal coverage there. Comparable clouds give a
  customer without a stream a record on a paid tier, or a push destination
  [7]. The trigger to build one is a Cloud customer with a compliance need.
- The database grows as it does today, plus one row per audit-log search
  (section 2).

**Implementation.** No setting and no API change. Two shipped strings that say
MCP refusals are recorded in the audit log become "streamed to stdout when it
is on", in all five locale files.

## 2. Annotation

We should keep the annotation as it is: `audit = true` marks the methods whose
calls the database stores. The store's rule does not vary by class or by
outcome, so a richer label would describe nothing the store uses.
SearchAuditLogs gains it, with its response omitted
from the row, so that access to the audit log is itself recorded;
ExportAuditLogs already carries it. The cost is one row per audit-log search,
and a log pipeline reads the stream rather than polling the search API.

**Implementation.** One audit option line on SearchAuditLogs and one omit
option on its response field.

## 3. Denials

- A refused call is one the method's permission check turned away, whether
  the access-control interceptor or the custom-auth handler made the check.
  Any other failure, a validation error or a failed connection test, is an
  ordinary call and follows section 1.
- Rules of other kinds are not permission checks and are not marked: a
  license gate, a workflow state, and ownership rules such as "only the
  assignee may do this".
- A marked refusal is stamped `WARNING`. Everything else stays `INFO`. The
  compliance reader filters on that, without joining on the status code.
- The two doors that refuse outside the connect chains, the `/mcp` connection
  and the OAuth consent page, refuse on the same ceiling verdict the gate
  uses and record only policy refusals, so their rows are `WARNING` too. The
  writer they share stamps it, so a third door cannot answer `INFO` for a
  refusal the gate calls `WARNING`.
- The validate-only rule stays: a validate-only call is skipped only when it
  succeeded.
- A failed login is a stored row today; a refused call is a new line in the
  stream. Both are the compliance reader's events, and neither is the
  administrator's.

**Implementation.** No API change. The mark is set where the access-control
interceptor answers permission-denied, at the MCP ceiling gate, at the
read-only SQL clamp, at the two MCP-origin guards in the issue and rollout
handlers, and at the custom-auth handler sites that make the check
themselves. Handler sites mark by building the refusal with
`permissionDeniedError` in `backend/api/v1`, so the mark cannot be forgotten separately
from the error; the two out-of-band doors mark through their shared writer.

## 4. Interceptor order

- The public chain becomes validate, auth, audit, then access control,
  matching the internal chain minus the gate. Without this the stream reader
  never sees a public-chain refusal, because it returns before the audit
  interceptor runs. Auth stays outside audit, since it populates the identity
  and workspace every row needs.
- The runtime flag is deleted. The write site makes two decisions: store if
  the method is audited and the call reached its handler; stream if stdout is
  on and the call was stored or refused by a permission check.
- No MCP exception. The MCP chain, whose enforcement shipped in 3.22.0 with
  the access-policy section of its settings page behind a dev gate until
  3.23.0, stops storing the ceiling-gate refusals it stores today; they stream
  instead. Clamp refusals run inside the audited Query handler and stay stored.

**Implementation.** No API change.

## Considered

- **Draft 3: a workspace policy with two switches, off by default.** We found
  no dedicated permission-denial audit toggle in the products reviewed, and a
  stored denial was never the admins' record [7].
- **Store every permission refusal.** One row per denied attempt in the
  metadata database with no bound but revocation, for an attempt that changed
  nothing an admin needs to trace.
- **A richer annotation: a class enum, or an explicit line on every method.**
  Two effective states without a policy, and about a hundred lines with no
  change in behavior. A class returns with read auditing, if it is built.
- **No annotation: a Go table keyed by method name, or customer-picked
  methods.** The labeling stays as it is, beside the other per-method
  options. Customer-written policy is Kubernetes' shape, and its own
  documentation warns that operators get it wrong [6].

## Not in this change

- Read auditing. The trigger is a customer asking; the shape is an option on
  the stream.
- A retention setting. Bytebase deletes no audit row; comparable products
  bound retention by default [7]. The lever under every rule.
- A stream for Cloud. Trigger and shape in section 1.
- A rate limit on unauthenticated calls. A login flood writes a row per
  attempt today, and the lockout bounds guesses, not rows.
- Exemptions by identity, GCP's exempted members [2], the damper for a
  retry-looping integration.
- Rejected credentials on API calls, refusals on the one streaming call, and
  a request-size cap.

## Reference

[1] [GCP Cloud Audit Logs: types, defaults, what cannot be disabled](https://docs.cloud.google.com/logging/docs/audit)

[2] [GCP Data Access audit log configuration: auditLogConfigs, logType, exemptedMembers](https://docs.cloud.google.com/logging/docs/audit/configure-data-access)

[3] [AWS CloudTrail: logging management events, read and write selectors](https://docs.aws.amazon.com/awscloudtrail/latest/userguide/logging-management-events-with-cloudtrail.html)

[4] [AWS CloudTrail: event history, the immutable 90-day baseline](https://docs.aws.amazon.com/awscloudtrail/latest/userguide/view-cloudtrail-events.html)

[5] [Azure Monitor: activity log, always collected, 90 days](https://learn.microsoft.com/en-us/azure/azure-monitor/essentials/activity-log)

[6] [Kubernetes: auditing and audit levels](https://kubernetes.io/docs/tasks/debug/debug-cluster/audit/)

[7] [Audit logs in comparable products, the research behind this draft](https://claude.ai/code/artifact/579af5ff-ff73-41ad-b446-9c94e9da7b46)
