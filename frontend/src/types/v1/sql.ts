import type { Code } from "@connectrpc/connect";
import type { QueryResponse } from "../proto-es/v1/sql_service_pb";

export interface SQLResultSetV1
  extends Omit<QueryResponse, "$typeName" | "appliedAccessGrant"> {
  error: string; // empty if no error occurred
  status?: Code;
  // The wire field is a required proto3 string (default ""), but adapter
  // sites (webTerminal AdminExecuteResponse spread, abort/error branches)
  // construct a SQLResultSetV1 before any grant context exists. Keeping it
  // optional here lets those sites omit it; treat absent or "" as no grant.
  appliedAccessGrant?: string;
}
