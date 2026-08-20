import type { Code } from "@connectrpc/connect";
import type { QueryResponse } from "../proto-es/v1/sql_service_pb";

export interface SQLResultSetV1
  extends Omit<
    QueryResponse,
    "$typeName" | "appliedAccessGrant" | "readOnlyEnforcement"
  > {
  error: string; // empty if no error occurred
  status?: Code;
  // The wire field is a required proto3 string (default ""), but adapter
  // sites (webTerminal AdminExecuteResponse spread, abort/error branches)
  // construct a SQLResultSetV1 before any grant context exists. Keeping it
  // optional here lets those sites omit it; treat absent or "" as no grant.
  appliedAccessGrant?: string;
  // Which enforcement held a query to reads, and only ever set for a session
  // whose MCP capability ceiling caps it at read-only. A person in the SQL
  // editor is never such a session, so this is always absent here; it is
  // optional for the same reason appliedAccessGrant is.
  readOnlyEnforcement?: QueryResponse["readOnlyEnforcement"];
}
