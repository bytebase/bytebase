# gh-ost Binlog Validation Error Design

## Context

Bytebase 3.16.0 can report `Binary logging is not enabled on this MySQL instance`
during gh-ost plan-save validation even when AWS RDS binary logging is enabled.
The current validator initializes `BinlogEnabled` to `false`, then returns early
if it cannot run `SHOW MASTER STATUS` or `SHOW BINARY LOG STATUS`. The
user-facing formatter checks `!BinlogEnabled` first, so an access or privilege
failure is misclassified as disabled binary logging.

The fix should preserve compatibility with older MySQL and managed MySQL
variants. Status access should point to `REPLICATION CLIENT`; gh-ost replication
privilege messaging should continue to accept older `REPLICATION SLAVE` wording
or an equivalent replication privilege.

## Goals

- Report binary logging as disabled only after `SELECT @@log_bin` succeeds and
  returns OFF or 0.
- Report binlog status access failures as access or privilege problems, not as
  disabled binary logging.
- Keep privilege guidance compatible with older MySQL/RDS installations.
- Add focused tests for customer-visible validation messages.

## Non-Goals

- Do not change gh-ost migration execution behavior.
- Do not require only modern privilege names such as `REPLICATION REPLICA`.
- Do not broaden the plan-check flow beyond gh-ost binlog prerequisite
  validation.

## Design

Add an explicit failure reason to `BinlogValidationResult`, implemented as an
unexported typed string with constants:

- `binlogStatusInaccessible`
- `binlogDisabled`
- `missingReplicationPrivilege`
- `unsupportedBinlogFormat`
- `validationQueryFailed`

`ValidateBinlogAccess()` should set the reason at the branch where validation
fails. `GetUserFriendlyError()` should switch on that reason instead of inferring
the cause from booleans.

Expected messages:

- Status access failure:
  `Cannot access binary log status. Ensure the Bytebase admin user has
  REPLICATION CLIENT privilege.`
- Disabled binlog:
  `Binary logging is not enabled on this MySQL instance.`
- Missing gh-ost replication privilege:
  mention `REPLICATION SLAVE` or equivalent replication privilege, preserving
  compatibility with older MySQL and RDS.
- Statement binlog format:
  keep the existing ROW/MIXED requirement message.

The validation should still try `SHOW MASTER STATUS` first for older versions
and fall back to `SHOW BINARY LOG STATUS` for MySQL 8.4+. The first failure
branch should include enough internal error detail for debugging, but the
customer-facing text should avoid claiming that binary logging is disabled unless
that was verified.

## Testing

Add `backend/component/ghost/validator_test.go` with table-driven tests for
`GetUserFriendlyError()` covering:

- binlog status inaccessible
- binary logging disabled
- missing replication privilege
- unsupported binlog format
- generic validation query failure

Keep this change focused on formatter tests and the direct reason assignment in
the validator branches. Do not add a new SQL mock dependency for this narrow
message classification fix.

Run:

```bash
gofmt -w backend/component/ghost/validator.go backend/component/ghost/validator_test.go
go test -v -count=1 ./backend/component/ghost
```

If implementation changes are made later, also run the repository-required
`golangci-lint run --allow-parallel-runners` before completion.
